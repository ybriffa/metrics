package metrics

import "testing"

func TestRegistryID(t *testing.T) {
	tests := []struct {
		name       string
		tags       map[string]string
		expectedID string
	}{
		{
			name: "test",
			tags: map[string]string{
				"a": "b",
				"c": "d",
			},
			expectedID: "test[a=b,c=d]",
		},
		{
			name:       "test",
			tags:       map[string]string{},
			expectedID: "test[]",
		},
	}

	for n, test := range tests {
		id := registryID(test.name, test.tags)
		if id != test.expectedID {
			t.Errorf("[test #%d] expected id %q, got %q", n, test.expectedID, id)
		}
	}
}

func TestRegistryName(t *testing.T) {
	tests := []struct {
		id           string
		expectedName string
	}{
		{
			id:           "test[a=b,c=d]",
			expectedName: "test",
		},
		{
			id:           "test",
			expectedName: "test",
		},
	}

	for n, test := range tests {
		name := registryName(test.id)
		if name != test.expectedName {
			t.Errorf("[test #%d] expected name %q, got %q", n, test.expectedName, name)
		}
	}
}
