package main

import (
	"encoding/json"
	"fmt"
	"testing"

	assert "github.com/stretchr/testify/require"
	"github.com/team-telnyx/telnyx-mock/spec"
)

var applicationFeeRefundCreateMethod *spec.Operation
var applicationFeeRefundGetMethod *spec.Operation
var chargeAllMethod *spec.Operation
var chargeCreateMethod *spec.Operation
var chargeGetMethod *spec.Operation
var customerDeleteMethod *spec.Operation
var invoicePayMethod *spec.Operation

// Try to avoid using the real spec as much as possible because it's more
// complicated and slower. A test spec is provided below. If you do use it,
// don't mutate it.
var realSpec spec.Spec
var realFixtures spec.Fixtures
var realComponentsForValidation *spec.ComponentsForValidation

var testSpec spec.Spec
var testFixtures spec.Fixtures

func init() {
	initRealSpec()
	initTestSpec()
}

func initRealSpec() {
	// Load the spec information from go-bindata
	data, err := Asset("openapi/openapi/spec3.json")
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, &realSpec)
	if err != nil {
		panic(err)
	}

	realComponentsForValidation =
		spec.GetComponentsForValidation(&realSpec.Components)

	// And do the same for fixtures
	data, err = Asset("openapi/openapi/fixtures3.json")
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, &realFixtures)
	if err != nil {
		panic(err)
	}
}

func initTestSpec() {
	// These are basically here to give us a URL to test against that has
	// multiple parameters in it.
	applicationFeeRefundCreateMethod = &spec.Operation{}
	applicationFeeRefundGetMethod = &spec.Operation{}

	chargeAllMethod = &spec.Operation{
		Parameters: []*spec.Parameter{
			{
				In:       spec.ParameterQuery,
				Name:     "limit",
				Required: false,
				Schema: &spec.Schema{
					Type: spec.TypeInteger,
				},
			},
		},
		Responses: map[spec.StatusCode]spec.Response{
			"200": {
				Ref: "#/components/responses/ChargeListResponse",
			},
		},
	}
	chargeCreateMethod = &spec.Operation{
		RequestBody: &spec.RequestBody{
			Content: map[string]spec.MediaType{
				"application/json": {
					Schema: &spec.Schema{
						AdditionalProperties: false,
						Properties: map[string]*spec.Schema{
							"amount": {
								Type: spec.TypeInteger,
							},
						},
						Required: []string{"amount"},
					},
				},
			},
		},
		Responses: map[spec.StatusCode]spec.Response{
			"200": {
				Ref: "#/components/responses/ChargeResponse",
			},
		},
	}
	chargeGetMethod = &spec.Operation{}

	customerDeleteMethod = &spec.Operation{
		RequestBody: &spec.RequestBody{
			Content: map[string]spec.MediaType{
				"application/json": {
					Schema: &spec.Schema{
						AdditionalProperties: false,
						Type:                 spec.TypeObject,
					},
				},
			},
		},
		Responses: map[spec.StatusCode]spec.Response{
			"200": {
				Ref: "#/components/responses/DeletedCustomerResponse",
			},
		},
	}

	// Here so we can test the relatively rare "action" operations (e.g.,
	// `POST` to `/pay` on an invoice).
	invoicePayMethod = &spec.Operation{}

	testFixtures =
		spec.Fixtures{
			Resources: map[spec.ResourceID]interface{}{
				spec.ResourceID("charge"): map[string]interface{}{
					"customer": "cus_123",
					"id":       "ch_123",
				},
				spec.ResourceID("customer"): map[string]interface{}{
					"id": "cus_123",
				},
				spec.ResourceID("deleted_customer"): map[string]interface{}{
					"deleted": true,
				},
			},
		}

	testSpec = spec.Spec{
		Components: spec.Components{
			Responses: map[string]*spec.Response{
				"DeletedCustomerResponse": {
					Content: map[string]spec.MediaType{
						"application/json": {
							Schema: &spec.Schema{
								Properties: map[string]*spec.Schema{
									"data": &spec.Schema{
										Ref: "#/components/schemas/deleted_customer",
									},
								},
							},
						},
					},
				},
				"ChargeResponse": {
					Content: map[string]spec.MediaType{
						"application/json": {
							Schema: &spec.Schema{
								Properties: map[string]*spec.Schema{
									"data": &spec.Schema{
										Ref: "#/components/schemas/charge",
									},
								},
							},
						},
					},
				},
				"ChargeListResponse": {
					Content: map[string]spec.MediaType{
						"application/json": {
							Schema: &spec.Schema{
								Properties: map[string]*spec.Schema{
									"data": &spec.Schema{
										Ref:  "",
										Type: "array",
										Items: &spec.Schema{
											Ref: "#/components/schemas/charge",
										},
									},
								},
							},
						},
					},
				},
			},
			Schemas: map[string]*spec.Schema{
				"charge": {
					Example: json.RawMessage(`{"id": "foo"}`),
					Type:    "object",
					Properties: map[string]*spec.Schema{
						"id": {Type: "string"},
						// Normally a customer ID, but expandable to a full
						// customer resource
						"customer": {
							AnyOf: []*spec.Schema{
								{Type: "string"},
								{Ref: "#/components/schemas/customer"},
							},
							XExpansionResources: &spec.ExpansionResources{
								OneOf: []*spec.Schema{
									{Ref: "#/components/schemas/customer"},
								},
							},
						},
					},
					XExpandableFields: &[]string{"customer"},
					XResourceID:       "charge",
				},
				"customer": {
					Example:     json.RawMessage(`{"id": "foo"}`),
					Type:        "object",
					XResourceID: "customer",
				},
				"deleted_customer": {
					Example: json.RawMessage(`{"id": "foo"}`),
					Properties: map[string]*spec.Schema{
						"deleted": {Type: "boolean"},
					},
					Type:        "object",
					XResourceID: "deleted_customer",
				},
			},
		},
		Paths: map[spec.Path]map[spec.HTTPVerb]*spec.Operation{
			spec.Path("/application_fees/{fee}/refunds"): {
				"get": applicationFeeRefundCreateMethod,
			},
			spec.Path("/application_fees/{fee}/refunds/{id}"): {
				"get": applicationFeeRefundGetMethod,
			},
			spec.Path("/charges"): {
				"get":  chargeAllMethod,
				"post": chargeCreateMethod,
			},
			spec.Path("/charges/{id}"): {
				"get": chargeGetMethod,
			},
			spec.Path("/customers/{id}"): {
				"delete": customerDeleteMethod,
			},
			spec.Path("/invoices/{id}/pay"): {
				"post": invoicePayMethod,
			},
		},
	}
}

func getDefaultOptions() *options {
	return &options{
		httpPort:  -1,
		httpsPort: -1,
		port:      -1,
	}
}

func TestCheckConflictingOptions(t *testing.T) {
	//
	// Valid sets of options (not exhaustive, but included quite a few standard invocations)
	//

	{
		options := getDefaultOptions()
		options.http = true

		err := options.checkConflictingOptions()
		assert.NoError(t, err)
	}

	{
		options := getDefaultOptions()
		options.https = true

		err := options.checkConflictingOptions()
		assert.NoError(t, err)
	}

	{
		options := getDefaultOptions()
		options.https = true
		options.port = 12111

		err := options.checkConflictingOptions()
		assert.NoError(t, err)
	}

	{
		options := getDefaultOptions()
		options.httpPort = 12111
		options.httpsPort = 12111

		err := options.checkConflictingOptions()
		assert.NoError(t, err)
	}

	{
		options := getDefaultOptions()
		options.httpUnixSocket = "/tmp/telnyx-mock.sock"
		options.httpsUnixSocket = "/tmp/telnyx-mock-secure.sock"

		err := options.checkConflictingOptions()
		assert.NoError(t, err)
	}

	//
	// Non-specific
	//

	{
		options := getDefaultOptions()
		options.port = 12111
		options.unixSocket = "/tmp/telnyx-mock.sock"

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please specify only one of -port or -unix"), err)
	}

	//
	// HTTP
	//

	{
		options := getDefaultOptions()
		options.http = true
		options.httpPort = 12111

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please don't specify -http when using -http-port or -http-unix"), err)
	}

	{
		options := getDefaultOptions()
		options.http = true
		options.httpUnixSocket = "/tmp/telnyx-mock.sock"

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please don't specify -http when using -http-port or -http-unix"), err)
	}

	{
		options := getDefaultOptions()
		options.port = 12111
		options.httpPort = 12111

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please don't specify -port or -unix when using -http-port or -http-unix"), err)
	}

	{
		options := getDefaultOptions()
		options.unixSocket = "/tmp/telnyx-mock.sock"
		options.httpUnixSocket = "/tmp/telnyx-mock.sock"

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please don't specify -port or -unix when using -http-port or -http-unix"), err)
	}

	{
		options := getDefaultOptions()
		options.httpPort = 12111
		options.httpUnixSocket = "/tmp/telnyx-mock.sock"

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please specify only one of -http-port or -http-unix"), err)
	}

	//
	// HTTPS
	//

	{
		options := getDefaultOptions()
		options.https = true
		options.httpsPort = 12111

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please don't specify -https when using -https-port or -https-unix"), err)
	}

	{
		options := getDefaultOptions()
		options.https = true
		options.httpsUnixSocket = "/tmp/telnyx-mock.sock"

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please don't specify -https when using -https-port or -https-unix"), err)
	}

	{
		options := getDefaultOptions()
		options.port = 12111
		options.httpsPort = 12111

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please don't specify -port or -unix when using -https-port or -https-unix"), err)
	}

	{
		options := getDefaultOptions()
		options.unixSocket = "/tmp/telnyx-mock.sock"
		options.httpsUnixSocket = "/tmp/telnyx-mock.sock"

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please don't specify -port or -unix when using -https-port or -https-unix"), err)
	}

	{
		options := getDefaultOptions()
		options.httpsPort = 12111
		options.httpsUnixSocket = "/tmp/telnyx-mock.sock"

		err := options.checkConflictingOptions()
		assert.Equal(t, fmt.Errorf("Please specify only one of -https-port or -https-unix"), err)
	}
}

// Specify :0 to ask the OS for a free port.
const freePort = 0

func TestOptionsGetHTTPListener(t *testing.T) {
	// Gets a listener when explicitly requested.
	{
		options := &options{
			httpPort: freePort,
		}
		listener, err := options.getHTTPListener()
		assert.NoError(t, err)
		assert.NotNil(t, listener)
		listener.Close()
	}

	// No listener when HTTPS is explicitly requested, but HTTP is not.
	{
		options := &options{
			httpPort:  -1, // Signals not specified
			httpsPort: freePort,
		}
		listener, err := options.getHTTPListener()
		assert.NoError(t, err)
		assert.Nil(t, listener)
	}

	// Activates on the default HTTP port if no other args provided.
	{
		options := &options{
			httpPortDefault: freePort,
		}
		listener, err := options.getHTTPListener()
		assert.NoError(t, err)
		assert.NotNil(t, listener)
		listener.Close()
	}
}

func TestOptionsGetNonSecureHTTPSListener(t *testing.T) {
	// Gets a listener when explicitly requested.
	{
		options := &options{
			httpsPort: freePort,
		}
		listener, err := options.getNonSecureHTTPSListener()
		assert.NoError(t, err)
		assert.NotNil(t, listener)
		listener.Close()
	}

	// No listener when HTTP is explicitly requested, but HTTPS is not.
	{
		options := &options{
			httpPort:  freePort,
			httpsPort: -1, // Signals not specified
		}
		listener, err := options.getNonSecureHTTPSListener()
		assert.NoError(t, err)
		assert.Nil(t, listener)
	}

	// No listener when HTTP is explicitly requested with the old `-port`
	// option.
	{
		options := &options{
			httpsPort: -1, // Signals not specified
			port:      freePort,
		}
		listener, err := options.getNonSecureHTTPSListener()
		assert.NoError(t, err)
		assert.Nil(t, listener)
	}

	// Activates on the default HTTPS port if no other args provided.
	{
		options := &options{
			httpsPortDefault: freePort,
		}
		listener, err := options.getNonSecureHTTPSListener()
		assert.NoError(t, err)
		assert.NotNil(t, listener)
		listener.Close()
	}
}
