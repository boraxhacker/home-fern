package ssm

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	awsssm "github.com/aws/aws-sdk-go-v2/service/ssm"
	awstypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"home-fern/internal/core"
	"strings"
	"time"
)

type ParamName string

func NewParamName(ptrname *string) (ParamName, core.ErrorCode) {

	name := aws.ToString(ptrname)

	chkname := strings.TrimPrefix(strings.ToLower(name), "/")
	if strings.HasPrefix(chkname, "aws") ||
		strings.HasPrefix(chkname, "ssm") ||
		strings.HasSuffix(chkname, "/") {

		return "", ErrInvalidName
	}

	return ParamName(name), core.ErrNone
}

func (p ParamName) asPathName() ParamName {

	if strings.HasPrefix(string(p), "/") {
		return p
	}

	return ParamName("/" + string(p))
}

func (p ParamName) asBeginsWithRegex() string {

	return "^" + string(p.asPathName()) + "/.*"
}

func (p ParamName) asEqualsRegex() string {

	return "^" + string(p.asPathName()) + "$"
}

type ParamPath string

func NewParamPath(ptrpath *string) (ParamPath, core.ErrorCode) {

	path := aws.ToString(ptrpath)

	if strings.HasPrefix(path, "/") {

		result := strings.TrimSuffix(path, "/")
		_, err := NewParamName(&result)
		if err != core.ErrNone {
			return "", err
		}

		return ParamPath(result), core.ErrNone
	}

	return "", ErrInvalidPath
}

func (p ParamPath) asRecursiveRegex() string {

	path := ParamPath(strings.TrimSuffix(string(p), "/"))

	return "^" + string(path) + "/.*"
}

func (p ParamPath) asOneLevelRegex() string {

	path := ParamPath(strings.TrimSuffix(string(p), "/"))

	return "^" + string(path) + "/[^/]+$"
}

type ParameterData struct {
	AllowedPattern   string
	DataType         string
	Description      string
	KeyId            string
	LastModifiedDate float64
	LastModifiedUser string
	Name             ParamName
	Policies         string
	Tags             []core.ResourceTag
	Tier             awstypes.ParameterTier
	Type             awstypes.ParameterType
	Value            string
	Version          int64
}

func NewParameterData(request *awsssm.PutParameterInput) (*ParameterData, core.ErrorCode) {

	paramName, err := NewParamName(request.Name)
	if err != core.ErrNone {
		return nil, err
	}

	result := ParameterData{
		AllowedPattern:   aws.ToString(request.AllowedPattern),
		DataType:         aws.ToString(request.DataType),
		Description:      aws.ToString(request.Description),
		LastModifiedDate: float64(time.Now().UnixNano()) / float64(time.Second),
		Name:             paramName.asPathName(),
		Policies:         aws.ToString(request.Policies),
		Tier:             request.Tier,
		Type:             request.Type,
		Value:            aws.ToString(request.Value),
	}

	if result.Tier == "" {
		result.Tier = awstypes.ParameterTierStandard
	}

	if result.DataType == "" {
		result.DataType = "text"
	}

	if result.Tier != awstypes.ParameterTierStandard &&
		result.Tier != awstypes.ParameterTierAdvanced &&
		result.Tier != awstypes.ParameterTierIntelligentTiering {

		return nil, ErrInvalidTier
	}

	if result.DataType != "text" &&
		result.DataType != "aws:ec2:image" &&
		result.DataType != "aws:ssm:integration" {

		return nil, ErrInvalidDataType
	}

	if result.Type != awstypes.ParameterTypeString &&
		result.Type != awstypes.ParameterTypeSecureString &&
		result.Type != awstypes.ParameterTypeStringList {

		return nil, ErrUnsupportedParameterType
	}

	for _, tag := range request.Tags {
		result.Tags = append(result.Tags,
			core.ResourceTag{Key: aws.ToString(tag.Key), Value: aws.ToString(tag.Value)})
	}

	return &result, core.ErrNone
}

type KeyFilterType string

const (
	NameKeyFilter     = "Name"
	TypeKeyFilter     = "Type"
	KeyIdFilter       = "KeyId"
	PathKeyFilter     = "Path"
	LabelKeyFilter    = "Label"
	TierKeyFilter     = "Tier"
	DataTypeKeyFilter = "DataType"
)

func (ktype KeyFilterType) isValid() bool {

	return ktype == NameKeyFilter || ktype == TypeKeyFilter || ktype == KeyIdFilter ||
		ktype == PathKeyFilter || ktype == LabelKeyFilter || ktype == TierKeyFilter ||
		ktype == DataTypeKeyFilter
}

type OptionFilterType string

const (
	EqualsOptionFilter     = "Equals"
	BeginsWithOptionFilter = "BeginsWith"
	RecursiveOptionFilter  = "Recursive"
	OneLevelOptionFilter   = "OneLevel"
)

func (ftype OptionFilterType) isValid() bool {

	return ftype == EqualsOptionFilter || ftype == BeginsWithOptionFilter ||
		ftype == RecursiveOptionFilter || ftype == OneLevelOptionFilter
}

type ParameterFilter struct {
	Key    KeyFilterType    `json:"Key"`
	Option OptionFilterType `json:"Option"`
	Values []string         `json:"Values"`
}

func NewParameterFilter(filter *awstypes.ParameterStringFilter) (*ParameterFilter, core.ErrorCode) {

	result := ParameterFilter{
		Key:    KeyFilterType(aws.ToString(filter.Key)),
		Option: OptionFilterType(aws.ToString(filter.Option)),
		Values: filter.Values,
	}

	if !result.Key.isValid() {
		return nil, ErrInvalidFilterKey
	}

	if !result.Option.isValid() {
		return nil, ErrInvalidFilterOption
	}

	if result.Key == PathKeyFilter {
		if result.Option != OneLevelOptionFilter && result.Option != RecursiveOptionFilter {
			return nil, ErrInvalidFilterOption
		}
	}

	if result.Key == NameKeyFilter {
		if result.Option != EqualsOptionFilter && result.Option != BeginsWithOptionFilter {
			return nil, ErrInvalidFilterOption
		}
	}

	return &result, core.ErrNone
}

type ParameterArnGenerator func(ParamName) string

type DescribeParameterItem struct {
	AllowedPattern   string                 `json:"AllowedPattern,omitempty"`
	ARN              string                 `json:"ARN"`
	DataType         string                 `json:"DataType"`
	Description      string                 `json:"Description,omitempty"`
	KeyId            string                 `json:"KeyId,omitempty"`
	LastModifiedDate float64                `json:"LastModifiedDate"`
	LastModifiedUser string                 `json:"LastModifiedUser"`
	Name             ParamName              `json:"Name"`
	Policies         string                 `json:"Policies,omitempty"`
	Tier             awstypes.ParameterTier `json:"Tier"`
	Type             awstypes.ParameterType `json:"Type"`
	Version          int64                  `json:"Version"`
}

type DescribeParametersResponse struct {
	NextToken  string                  `json:"NextToken,omitempty"`
	Parameters []DescribeParameterItem `json:"Parameters"`
}

type GetParameterItem struct {
	ARN              string                 `json:"ARN"`
	DataType         string                 `json:"DataType"`
	LastModifiedDate float64                `json:"LastModifiedDate"`
	Name             ParamName              `json:"Name"`
	Selector         string                 `json:"Selector,omitempty"`
	SourceResult     string                 `json:"SourceResult,omitempty"`
	Type             awstypes.ParameterType `json:"Type"`
	Value            string                 `json:"Value"`
	Version          int64                  `json:"Version"`
}

type GetParametersResponse struct {
	InvalidParameters []string           `json:"InvalidParameters"`
	Parameters        []GetParameterItem `json:"Parameters"`
}

type GetParametersByPathResponse struct {
	NextToken  string             `json:"NextToken"`
	Parameters []GetParameterItem `json:"Parameters"`
}

type GetParameterResponse struct {
	Parameter *GetParameterItem `json:"Parameter"`
}

func (param *ParameterData) toGetParameterItem(arnGenerator ParameterArnGenerator) *GetParameterItem {

	return &GetParameterItem{
		ARN:              arnGenerator(param.Name),
		DataType:         param.DataType,
		LastModifiedDate: param.LastModifiedDate,
		Name:             param.Name,
		Type:             param.Type,
		Value:            param.Value,
		Version:          param.Version,
	}
}

func (param *ParameterData) toDescribeParameterItem(arnGenerator ParameterArnGenerator) *DescribeParameterItem {

	return &DescribeParameterItem{
		AllowedPattern:   param.AllowedPattern,
		ARN:              arnGenerator(param.Name),
		DataType:         param.DataType,
		Description:      param.Description,
		KeyId:            param.KeyId,
		LastModifiedDate: param.LastModifiedDate,
		LastModifiedUser: param.LastModifiedUser,
		Name:             param.Name,
		Tier:             param.Tier,
		Type:             param.Type,
		Version:          param.Version,
	}
}
