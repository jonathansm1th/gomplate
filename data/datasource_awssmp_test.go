// +build !windows

package data

import (
	"net/url"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/stretchr/testify/assert"
)

// DummyParamGetter - test doubles
type DummyParamGetter struct {
	t *testing.T
	param *ssm.Parameter
	params []*ssm.Parameter
	err awserr.Error
	mockGetParameter func(*ssm.GetParameterInput) (*ssm.GetParameterOutput, error)
	mockGetParametersByPathPages func(*ssm.GetParametersByPathInput, func(*ssm.GetParametersByPathOutput, bool) bool) error
}

func (d DummyParamGetter) GetParameter(input *ssm.GetParameterInput) (*ssm.GetParameterOutput, error) {
	assert.Nil(d.t, d.mockGetParametersByPathPages,
		"Looks like you expected the mock func to be called...")
	if d.mockGetParameter != nil {
		output, err := d.mockGetParameter(input)
		return output, err
	}
	if d.err != nil {
		return nil, d.err
	}
	assert.NotNil(d.t, d.param, "Must provide a param if no error!")
	return &ssm.GetParameterOutput{
		Parameter: d.param,
	}, nil
}

func (d DummyParamGetter) GetParametersByPathPages(input *ssm.GetParametersByPathInput,
		fn func(*ssm.GetParametersByPathOutput, bool) bool) error {
	assert.Nil(d.t, d.mockGetParameter)
	if d.mockGetParametersByPathPages != nil {
		return d.mockGetParametersByPathPages(input, fn)
	}
	if d.err != nil {
		return d.err
	}
	if d.params != nil {
		fn(&ssm.GetParametersByPathOutput{
			// NextToken: aws.String("dummy"),
			Parameters: d.params,
		}, false)
	}
	return nil
}

func simpleAWSSourceHelper(dummy AWSSMPGetter) *Source {
	return &Source{
		Alias: "foo",
		URL:   &url.URL{
			Scheme: "aws+smp",
			Path:   "/foo",
		},
		ASMPG: dummy,
	}
}

func TestAWSSMP_ParseArgsSimple(t *testing.T) {
	paramPath, mode, err := parseAWSSMPArgs("noddy")
	assert.Equal(t, "noddy", paramPath)
	assert.Equal(t, "", mode)
	assert.Nil(t, err)
}

func TestAWSSMP_ParseArgsAppend(t *testing.T) {
	paramPath, mode, err := parseAWSSMPArgs("base", "extra", modeRecursive)
	assert.Equal(t, "base/extra", paramPath)
	assert.Equal(t, modeRecursive, mode)
	assert.Nil(t, err)
}

func TestAWSSMP_ParseArgsAppend2(t *testing.T) {
	paramPath, mode, err := parseAWSSMPArgs("/foo/", "/extra", modeOneLevel)
	assert.Equal(t, "/foo/extra", paramPath)
	assert.Equal(t, modeOneLevel, mode)
	assert.Nil(t, err)
}

func TestAWSSMP_ParseArgsBadMode(t *testing.T) {
	_, _, err := parseAWSSMPArgs("base", "extra", "fdsfds")
	assert.Error(t, err)
}

func TestAWSSMP_ParseArgsTooMany(t *testing.T) {
	_, _, err := parseAWSSMPArgs("base", "extra", modeRecursive, modeOneLevel)
	assert.Error(t, err)
}

func TestAWSSMP_GetParameterSetup(t *testing.T) {
	calledOk := false
	s := simpleAWSSourceHelper(DummyParamGetter{
			t: t,
			mockGetParameter: func(input *ssm.GetParameterInput) (*ssm.GetParameterOutput, error) {
				assert.Equal(t, "/foo/bar", *input.Name)
				assert.True(t, *input.WithDecryption)
				calledOk = true
				return &ssm.GetParameterOutput{
					Parameter: &ssm.Parameter{},
				}, nil
			},
		})

	_, err := readAWSSMP(s, "/bar")
	assert.True(t, calledOk)
	assert.Nil(t, err)
}

func TestAWSSMP_GetParameterValidOutput(t *testing.T) {
	s := simpleAWSSourceHelper(DummyParamGetter{
			t: t,
			param: &ssm.Parameter{
				Name: aws.String("/foo"),
				Type: aws.String("String"),
				Value: aws.String("val"),
				Version: aws.Int64(1),
			},
		})

	output, err := readAWSSMP(s, "")
	assert.Nil(t, err)
	expected := "{\"Name\":\"/foo\",\"Type\":\"String\",\"Value\":\"val\",\"Version\":1}"
	assert.Equal(t, []byte(expected), output)
	assert.Equal(t, json_mimetype, s.Type)
}

func TestAWSSMP_GetParameterMissing(t *testing.T) {
	expectedErr := awserr.New("ParameterNotFound", "Test of error message", nil)
	s := simpleAWSSourceHelper(DummyParamGetter{
			t: t,
			err: expectedErr,
		})

	defer restoreLogFatalf()
	setupMockLogFatalf()
	assert.Panics(t, func() {
		readAWSSMP(s, "")
	})
	assert.Contains(t, spyLogFatalfMsg, "Test of error message")
}

func TestAWSSMP_GetParametersByPathValidOutput(t *testing.T) {
	s := simpleAWSSourceHelper(DummyParamGetter{
			t: t,
			params: []*ssm.Parameter{&ssm.Parameter{
				Name: aws.String("/foo"),
				Type: aws.String("String"),
				Value: aws.String("val"),
				Version: aws.Int64(1),
			}},
		})

	output, err := readAWSSMP(s, "", modeOneLevel)
	assert.Nil(t, err)
	expected := "[{\"Name\":\"/foo\",\"Type\":\"String\",\"Value\":\"val\",\"Version\":1}]"
	assert.Equal(t, []byte(expected), output)
	assert.Equal(t, json_array_mimetype, s.Type)
}

func TestAWSSMP_GetParametersByPathError(t *testing.T) {
	s := simpleAWSSourceHelper(DummyParamGetter{
			t: t,
			err: awserr.New("ParameterNotFound", "Parameters error message", nil),
		})

	defer restoreLogFatalf()
	setupMockLogFatalf()
	assert.Panics(t, func() {
		readAWSSMP(s, "", modeOneLevel)
	})
	assert.Contains(t, spyLogFatalfMsg, "Parameters error message")
}
