package ssm

import (
	"fmt"
	"home-fern/internal/core"
	"home-fern/internal/datastore"
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

func NewService(fernConfig *core.FernConfig, accountId string, ds *datastore.Datastore) *Service {

	dataStore := newDataStore(ds, fernConfig.Keys)

	result := Service{
		region:    fernConfig.Region,
		accountId: accountId,
		dataStore: dataStore,
	}

	return &result
}

func (service *Service) DeleteParameter(
	request *awsssm.DeleteParameterInput) (*awsssm.DeleteParameterOutput, error) {

	paramName, err := NewParamName(request.Name)
	if err != nil {
		return nil, err
	}

	err = service.dataStore.delete(string(paramName.asPathName()))
	if err != nil {
		return nil, err
	}

	return &awsssm.DeleteParameterOutput{}, nil
}

func (service *Service) DeleteParameters(
	request *awsssm.DeleteParametersInput) (*awsssm.DeleteParametersOutput, error) {

	var response awsssm.DeleteParametersOutput
	for _, name := range request.Names {

		paramName, err := NewParamName(&name)
		if err != nil {

			response.InvalidParameters = append(response.InvalidParameters, name)

		} else {

			err := service.dataStore.delete(string(paramName.asPathName()))
			if err == nil {

				response.DeletedParameters = append(response.DeletedParameters, name)

			} else {

				response.InvalidParameters = append(response.InvalidParameters, name)
			}
		}
	}

	return &response, nil
}

func (service *Service) DescribeParameters(
	request *awsssm.DescribeParametersInput) (*DescribeParametersResponse, error) {

	// TODO incomplete implementation

	var parameters []ParameterData

	var filters []string

	for _, awsfilter := range request.ParameterFilters {

		filter, err := NewParameterFilter(&awsfilter)
		if err != nil {
			return nil, err
		}

		if (filter.Key == NameKeyFilter && filter.Option == EqualsOptionFilter) ||
			(filter.Key == NameKeyFilter && filter.Option == BeginsWithOptionFilter) {

			for _, value := range filter.Values {

				paramName, err := NewParamName(&value)
				if err != nil {
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
				if err != nil {
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
	if err != nil {

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

	return &response, nil
}

func (service *Service) GetParameter(
	request *awsssm.GetParameterInput) (*GetParameterResponse, error) {

	result, err := service.getParameterByName(
		aws.ToString(request.Name), aws.ToBool(request.WithDecryption))
	if err != nil {
		return nil, err
	}

	response := GetParameterResponse{
		Parameter: result.toGetParameterItem(service.createParameterArn),
	}

	return &response, nil
}

func (service *Service) GetParameters(
	request *awsssm.GetParametersInput) (*GetParametersResponse, error) {

	var response GetParametersResponse
	for _, name := range request.Names {

		param, err := service.getParameterByName(name, aws.ToBool(request.WithDecryption))
		if err == nil {
			item := param.toGetParameterItem(service.createParameterArn)
			response.Parameters = append(response.Parameters, *item)
		} else {
			response.InvalidParameters = append(response.InvalidParameters, name)
		}
	}

	return &response, nil
}

func (service *Service) GetParametersByPath(
	request *awsssm.GetParametersByPathInput) (*GetParametersByPathResponse, error) {

	// TODO incomplete implementation

	paramPath, err := NewParamPath(request.Path)
	if err != nil {
		return nil, err
	}

	for _, filter := range request.ParameterFilters {
		_, err := NewParameterFilter(&filter)
		if err != nil {
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
	if err != nil {

		return nil, err
	}

	var response GetParametersByPathResponse
	if nextToken != "" {
		response.NextToken = nextToken
	}

	for _, param := range parameters {

		if aws.ToBool(request.WithDecryption) && param.Type == awstypes.ParameterTypeSecureString {

			decryptedValue, err := service.dataStore.decrypt(param.Value, param.KeyId)
			if err != nil {
				return nil, err
			}

			param.Value = decryptedValue
		}

		response.Parameters = append(response.Parameters,
			*param.toGetParameterItem(service.createParameterArn))
	}

	return &response, nil
}

func (service *Service) PutParameter(
	creds *aws.Credentials, request *awsssm.PutParameterInput) (*awsssm.PutParameterOutput, error) {

	// tags from previous version are kept and request params are ignored
	return service.persistParameter(creds, request, false)
}

func (service *Service) LogKeys(writer io.Writer) error {
	return service.dataStore.logKeys(writer)
}

func (service *Service) AddTagsToResource(
	request *awsssm.AddTagsToResourceInput) (*awsssm.AddTagsToResourceOutput, error) {

	var response awsssm.AddTagsToResourceOutput
	if request.ResourceType == awstypes.ResourceTypeForTaggingParameter {

		param, err := service.getParameterByName(aws.ToString(request.ResourceId), false)
		if err != nil {
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
		if err != nil {
			return nil, err
		}
	}

	return &response, nil
}

func (service *Service) RemoveTagsFromResource(
	request *awsssm.RemoveTagsFromResourceInput) (*awsssm.RemoveTagsFromResourceOutput, error) {

	var response awsssm.RemoveTagsFromResourceOutput
	if request.ResourceType == awstypes.ResourceTypeForTaggingParameter {

		param, err := service.getParameterByName(aws.ToString(request.ResourceId), false)
		if err != nil {
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
		if err != nil {
			return nil, err
		}
	}

	return &response, nil
}

func (service *Service) ListTagsForResource(
	request *awsssm.ListTagsForResourceInput) (*awsssm.ListTagsForResourceOutput, error) {

	var response awsssm.ListTagsForResourceOutput
	if request.ResourceType == awstypes.ResourceTypeForTaggingParameter {

		param, err := service.getParameterByName(aws.ToString(request.ResourceId), false)
		if err != nil {
			return nil, err
		}

		for _, paramTag := range param.Tags {

			response.TagList = append(response.TagList,
				awstypes.Tag{Key: aws.String(paramTag.Key), Value: aws.String(paramTag.Value)})
		}
	}

	return &response, nil
}

func (service *Service) GetAllParameters() ([]ParameterData, error) {

	var allParams []ParameterData
	var nextToken string

	for {
		params, token, err := service.dataStore.findParametersByKey([]string{".*"}, 50, nextToken)
		if err != nil {
			return nil, err
		}

		for i := range params {
			if params[i].Type == awstypes.ParameterTypeSecureString {
				decryptedValue, err := service.dataStore.decrypt(params[i].Value, params[i].KeyId)
				if err != nil {
					return nil, err
				}
				params[i].Value = decryptedValue
			}
		}

		allParams = append(allParams, params...)
		nextToken = token

		if nextToken == "" {
			break
		}
	}

	return allParams, nil
}

func (service *Service) DeleteAllData() error {
	return service.dataStore.deleteAll()
}

func (service *Service) ImportParameters(
	creds *aws.Credentials, parameters []ParameterData, overwrite bool) ([]string, error) {

	var failures []string

	for _, param := range parameters {

		var tags []awstypes.Tag
		for _, t := range param.Tags {
			tags = append(tags, awstypes.Tag{
				Key:   aws.String(t.Key),
				Value: aws.String(t.Value),
			})
		}

		putInput := &awsssm.PutParameterInput{
			Name:           aws.String(string(param.Name)),
			Value:          aws.String(param.Value),
			Type:           param.Type,
			Overwrite:      aws.Bool(overwrite),
			Description:    aws.String(param.Description),
			AllowedPattern: aws.String(param.AllowedPattern),
			KeyId:          aws.String(param.KeyId),
			Policies:       aws.String(param.Policies),
			Tier:           param.Tier,
			DataType:       aws.String(param.DataType),
			Tags:           tags,
		}

		// overwrite = true => we want our tags to persist and ignore previous version tags
		_, err := service.persistParameter(creds, putInput, overwrite)
		if err != nil {
			failures = append(failures, string(param.Name))
			continue
		}
	}

	return failures, nil
}

func (service *Service) createUserArn(creds *aws.Credentials) string {

	return fmt.Sprintf("arn:aws:iam::%s:user/%s", service.accountId, creds.Source)
}

func (service *Service) getParameterByName(name string, withDecryption bool) (*ParameterData, error) {

	paramName, err := NewParamName(&name)
	if err != nil {
		return nil, err
	}

	result, err := service.dataStore.getParameter(string(paramName.asPathName()))
	if err != nil {
		return nil, err
	}

	// always stored as path but if requested by name then return the name
	result.Name = paramName

	if result.Type == "SecureString" && withDecryption {

		decryptedValue, err := service.dataStore.decrypt(result.Value, result.KeyId)
		if err != nil {
			return nil, err
		}

		result.Value = decryptedValue
	}

	return result, nil
}

func (service *Service) createParameterArn(name ParamName) string {

	return fmt.Sprintf("arn:aws:ssm:%s:%s:parameter/%s",
		service.region, service.accountId, strings.TrimPrefix(string(name), "/"))
}

func (service *Service) persistParameter(
	creds *aws.Credentials, request *awsssm.PutParameterInput, skipTagCopy bool) (*awsssm.PutParameterOutput, error) {

	param, err := NewParameterData(request)
	if err != nil {
		return nil, err
	}

	param.LastModifiedUser = service.createUserArn(creds)

	if param.Type == awstypes.ParameterTypeSecureString {

		if param.KeyId == "" {
			param.KeyId = "alias/" + service.dataStore.keys[0].Alias
		}

		encryptedValue, err := service.dataStore.encrypt(param.Value, param.KeyId)
		if err != nil {
			return nil, err
		}

		param.Value = encryptedValue
	}

	newVersion, err := service.dataStore.putParameter(
		string(param.Name), param, aws.ToBool(request.Overwrite), skipTagCopy)

	if err != nil {

		return nil, err
	}

	return &awsssm.PutParameterOutput{Tier: awstypes.ParameterTierStandard, Version: newVersion}, nil
}
