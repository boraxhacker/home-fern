package ssm

import (
	"fmt"
	"home-fern/internal/core"
	"io"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsssm "github.com/aws/aws-sdk-go-v2/service/ssm"
	awstypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type Service struct {
	dataStore *dataStore
	accountId string
	region    string
}

func NewService(fernConfig *core.FernConfig, accountId string, dataPath string) *Service {

	databasePath := dataPath + "/ssm"
	dataStore := newDataStore(databasePath, fernConfig.Keys)

	result := Service{
		region:    fernConfig.Region,
		accountId: accountId,
		dataStore: dataStore,
	}

	return &result
}

func (service *Service) Close() {

	if service.dataStore != nil {
		service.dataStore.Close()
	}
}

func (service *Service) DeleteParameter(
	request *awsssm.DeleteParameterInput) (*awsssm.DeleteParameterOutput, core.ErrorCode) {

	paramName, err := NewParamName(request.Name)
	if err != core.ErrNone {
		return nil, ErrInvalidName
	}

	err = service.dataStore.delete(string(paramName.asPathName()))
	if err != core.ErrNone {
		return nil, err
	}

	return &awsssm.DeleteParameterOutput{}, core.ErrNone
}

func (service *Service) DeleteParameters(
	request *awsssm.DeleteParametersInput) (*awsssm.DeleteParametersOutput, core.ErrorCode) {

	var response awsssm.DeleteParametersOutput
	for _, name := range request.Names {

		paramName, err := NewParamName(&name)
		if err != core.ErrNone {

			response.InvalidParameters = append(response.InvalidParameters, name)

		} else {

			err := service.dataStore.delete(string(paramName.asPathName()))
			if err == core.ErrNone {

				response.DeletedParameters = append(response.DeletedParameters, name)

			} else {

				response.InvalidParameters = append(response.InvalidParameters, name)
			}
		}
	}

	return &response, core.ErrNone
}

func (service *Service) DescribeParameters(
	request *awsssm.DescribeParametersInput) (*DescribeParametersResponse, core.ErrorCode) {

	// TODO incomplete implementation

	var parameters []ParameterData

	var filters []string

	for _, awsfilter := range request.ParameterFilters {

		filter, err := NewParameterFilter(&awsfilter)
		if err != core.ErrNone {
			return nil, err
		}

		if (filter.Key == NameKeyFilter && filter.Option == EqualsOptionFilter) ||
			(filter.Key == NameKeyFilter && filter.Option == BeginsWithOptionFilter) {

			for _, value := range filter.Values {

				paramName, err := NewParamName(&value)
				if err != core.ErrNone {
					return nil, err
				}

				if filter.Option == BeginsWithOptionFilter {

					filters = append(filters, paramName.asBeginsWithRegex())
				} else {

					filters = append(filters, paramName.asEqualsRegex())
				}
			}
		}

		if (filter.Key == PathKeyFilter && filter.Option == RecursiveOptionFilter) ||
			(filter.Key == PathKeyFilter && filter.Option == OneLevelOptionFilter) {

			for _, value := range filter.Values {

				paramPath, err := NewParamPath(&value)
				if err != core.ErrNone {
					return nil, err
				}

				if filter.Option == RecursiveOptionFilter {
					filters = append(filters, paramPath.asRecursiveRegex())
				} else {
					filters = append(filters, paramPath.asOneLevelRegex())
				}
			}
		}
	}

	maxResults := 50
	if request.MaxResults != nil && *request.MaxResults > 0 && *request.MaxResults < int32(maxResults) {

		maxResults = int(aws.ToInt32(request.MaxResults))
	}

	parameters, nextToken, err :=
		service.dataStore.findParametersByKey(filters, maxResults, aws.ToString(request.NextToken))
	if err != core.ErrNone {

		return nil, err
	}

	var response DescribeParametersResponse
	if nextToken != "" {
		response.NextToken = nextToken
	}

	for _, param := range parameters {

		response.Parameters = append(response.Parameters,
			*param.toDescribeParameterItem(service.createParameterArn))
	}

	return &response, core.ErrNone
}

func (service *Service) GetParameter(
	request *awsssm.GetParameterInput) (*GetParameterResponse, core.ErrorCode) {

	result, err := service.getParameterByName(
		aws.ToString(request.Name), aws.ToBool(request.WithDecryption))
	if err != core.ErrNone {
		return nil, err
	}

	response := GetParameterResponse{
		Parameter: result.toGetParameterItem(service.createParameterArn),
	}

	return &response, core.ErrNone
}

func (service *Service) GetParameters(
	request *awsssm.GetParametersInput) (*GetParametersResponse, core.ErrorCode) {

	var response GetParametersResponse
	for _, name := range request.Names {

		param, err := service.getParameterByName(name, aws.ToBool(request.WithDecryption))
		if err == core.ErrNone {
			item := param.toGetParameterItem(service.createParameterArn)
			response.Parameters = append(response.Parameters, *item)
		} else {
			response.InvalidParameters = append(response.InvalidParameters, name)
		}
	}

	return &response, core.ErrNone
}

func (service *Service) GetParametersByPath(
	request *awsssm.GetParametersByPathInput) (*GetParametersByPathResponse, core.ErrorCode) {

	// TODO incomplete implementation

	paramPath, err := NewParamPath(request.Path)
	if err != core.ErrNone {
		return nil, err
	}

	for _, filter := range request.ParameterFilters {
		_, err := NewParameterFilter(&filter)
		if err != core.ErrNone {
			return nil, err
		}
	}

	var filters []string
	if aws.ToBool(request.Recursive) {
		filters = append(filters, paramPath.asRecursiveRegex())
	} else {
		filters = append(filters, paramPath.asOneLevelRegex())
	}

	maxResults := 10
	if request.MaxResults != nil && *request.MaxResults > 0 && *request.MaxResults < int32(maxResults) {

		maxResults = int(aws.ToInt32(request.MaxResults))
	}

	parameters, nextToken, err :=
		service.dataStore.findParametersByKey(filters, maxResults, aws.ToString(request.NextToken))
	if err != core.ErrNone {

		return nil, err
	}

	var response GetParametersByPathResponse
	if nextToken != "" {
		response.NextToken = nextToken
	}

	for _, param := range parameters {

		if aws.ToBool(request.WithDecryption) && param.Type == awstypes.ParameterTypeSecureString {

			decryptedValue, err := service.dataStore.decrypt(param.Value, param.KeyId)
			if err != core.ErrNone {
				return nil, ErrInvalidKeyId
			}

			param.Value = decryptedValue
		}

		response.Parameters = append(response.Parameters,
			*param.toGetParameterItem(service.createParameterArn))
	}

	return &response, core.ErrNone
}

func (service *Service) PutParameter(
	creds *aws.Credentials, request *awsssm.PutParameterInput) (*awsssm.PutParameterOutput, core.ErrorCode) {

	param, err := NewParameterData(request)
	if err != core.ErrNone {
		return nil, err
	}

	param.LastModifiedUser = service.createUserArn(creds)

	if param.Type == awstypes.ParameterTypeSecureString {

		if param.KeyId == "" {
			param.KeyId = "alias/" + service.dataStore.keys[0].Alias
		}

		encryptedValue, err := service.dataStore.encrypt(param.Value, param.KeyId)
		if err != core.ErrNone {
			return nil, err
		}

		param.Value = encryptedValue
	}

	newVersion, err := service.dataStore.putParameter(
		string(param.Name), param, aws.ToBool(request.Overwrite), false)

	if err != core.ErrNone {

		return nil, err
	}

	return &awsssm.PutParameterOutput{Tier: awstypes.ParameterTierStandard, Version: newVersion}, core.ErrNone
}

func (service *Service) LogKeys(writer io.Writer) error {

	return core.LogKeys(service.dataStore.db, writer)
}

func (service *Service) AddTagsToResource(
	request *awsssm.AddTagsToResourceInput) (*awsssm.AddTagsToResourceOutput, core.ErrorCode) {

	var response awsssm.AddTagsToResourceOutput
	if request.ResourceType == awstypes.ResourceTypeForTaggingParameter {

		param, err := service.getParameterByName(aws.ToString(request.ResourceId), false)
		if err != core.ErrNone {
			return nil, err
		}

		for _, tag := range request.Tags {

			reqTagKey := aws.ToString(tag.Key)
			reqTagValue := aws.ToString(tag.Value)

			found := false
			for idx, _ := range param.Tags {

				if param.Tags[idx].Key == reqTagKey {
					param.Tags[idx].Value = reqTagValue
					found = true
					break
				}
			}

			if !found {

				param.Tags = append(param.Tags,
					core.ResourceTag{Key: reqTagKey, Value: reqTagValue})
			}
		}

		_, err = service.dataStore.putParameter(string(param.Name), param, true, true)
		if err != core.ErrNone {
			return nil, err
		}
	}

	return &response, core.ErrNone
}

func (service *Service) RemoveTagsFromResource(
	request *awsssm.RemoveTagsFromResourceInput) (*awsssm.RemoveTagsFromResourceOutput, core.ErrorCode) {

	var response awsssm.RemoveTagsFromResourceOutput
	if request.ResourceType == awstypes.ResourceTypeForTaggingParameter {

		param, err := service.getParameterByName(aws.ToString(request.ResourceId), false)
		if err != core.ErrNone {
			return nil, err
		}

		for _, tag := range request.TagKeys {

			for i, paramTag := range param.Tags {

				if paramTag.Key == tag {
					param.Tags = slices.Delete(param.Tags, i, i+1)
					break
				}
			}
		}

		_, err = service.dataStore.putParameter(string(param.Name), param, true, true)
		if err != core.ErrNone {
			return nil, err
		}
	}

	return &response, core.ErrNone
}

func (service *Service) ListTagsForResource(
	request *awsssm.ListTagsForResourceInput) (*awsssm.ListTagsForResourceOutput, core.ErrorCode) {

	var response awsssm.ListTagsForResourceOutput
	if request.ResourceType == awstypes.ResourceTypeForTaggingParameter {

		param, err := service.getParameterByName(aws.ToString(request.ResourceId), false)
		if err != core.ErrNone {
			return nil, err
		}

		for _, paramTag := range param.Tags {

			response.TagList = append(response.TagList,
				awstypes.Tag{Key: aws.String(paramTag.Key), Value: aws.String(paramTag.Value)})
		}
	}

	return &response, core.ErrNone
}

func (service *Service) createUserArn(creds *aws.Credentials) string {

	return fmt.Sprintf("arn:aws:iam::%s:user/%s", service.accountId, creds.Source)
}

func (service *Service) getParameterByName(name string, withDecryption bool) (*ParameterData, core.ErrorCode) {

	paramName, err := NewParamName(&name)
	if err != core.ErrNone {
		return nil, ErrInvalidName
	}

	result, err := service.dataStore.getParameter(string(paramName.asPathName()))
	if err != core.ErrNone {
		return nil, err
	}

	// always stored as path but if requested by name then return the name
	result.Name = paramName

	if result.Type == "SecureString" && withDecryption {

		decryptedValue, err := service.dataStore.decrypt(result.Value, result.KeyId)
		if err != core.ErrNone {
			return nil, err
		}

		result.Value = decryptedValue
	}

	return result, core.ErrNone
}

func (service *Service) createParameterArn(name ParamName) string {

	return fmt.Sprintf("arn:aws:ssm:%s:%s:parameter/%s",
		service.region, service.accountId, strings.TrimPrefix(string(name), "/"))
}
