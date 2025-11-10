package examples

import (
	"fmt"
	"testing"
)

func TestGetAuthCode(t *testing.T) {
	code, err := getAuthCode("https://example.com/?state=state-token&code=4%2F0ASVgi3LXa4gceyqWJO3YuJm1dxZaFuc3R6V9zbhTfETTWJqR5APeS7xl8BITc-j3drVMUQ&scope=https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fyoutube.readonly")
	if err != nil {
		t.Error(err)
	}

	expectedCode := "REDACTED"
	if code != expectedCode {
		t.Fatalf(`wait %s but was %s`, expectedCode, code)
	}
}

func TestGetAuthCode_Errors(t *testing.T) {
	// Define test cases with inputs and expected results
	// Remove cases with nil error
	testCases := []struct {
		name string // Test case name
		url  string // url to pass into getAuthCode
		err  error  // Expected error
	}{
		{
			name: "Missing code query parameter",
			url:  "https://example.com/?state=state-token",
			err:  fmt.Errorf("no code param: https://example.com/?state=state-token"),
		},
		{
			name: "Empty url",
			url:  "",
			err:  fmt.Errorf("url is empty"),
		},
		{
			name: "empty code value",
			url:  "https//example.com/invalid?state=state-token&code=",
			err:  fmt.Errorf("no code value: https//example.com/invalid?state=state-token&code="),
		},
	}

	// Iterate over the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := getAuthCode(tc.url)

			// Verify error condition matches the expected behavior
			if (tc.err == nil && err != nil) || (tc.err != nil && err == nil) {
				t.Errorf("expected error %v but got %v", tc.err, err)
			}
			if tc.err != nil && err != nil && tc.err.Error() != err.Error() {
				t.Errorf("expected error message %q but got %q", tc.err.Error(), err.Error())
			}
		})
	}
}
