// Code generated by go-swagger; DO NOT EDIT.

package groups

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new groups API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for groups API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientService is the interface for Client methods
type ClientService interface {
	AddNewGroup(params *AddNewGroupParams, authInfo runtime.ClientAuthInfoWriter) (*AddNewGroupOK, error)

	AddPoolsByGroupID(params *AddPoolsByGroupIDParams, authInfo runtime.ClientAuthInfoWriter) (*AddPoolsByGroupIDOK, error)

	DeleteGroup(params *DeleteGroupParams, authInfo runtime.ClientAuthInfoWriter) error

	DeletePoolsByGroupID(params *DeletePoolsByGroupIDParams, authInfo runtime.ClientAuthInfoWriter) (*DeletePoolsByGroupIDOK, error)

	GetGroupDescriptionByID(params *GetGroupDescriptionByIDParams, authInfo runtime.ClientAuthInfoWriter) (*GetGroupDescriptionByIDOK, error)

	GetGroupIDByName(params *GetGroupIDByNameParams, authInfo runtime.ClientAuthInfoWriter) (*GetGroupIDByNameOK, error)

	GetPoolsByGroupID(params *GetPoolsByGroupIDParams, authInfo runtime.ClientAuthInfoWriter) (*GetPoolsByGroupIDOK, error)

	ReplacePoolsByGroupID(params *ReplacePoolsByGroupIDParams, authInfo runtime.ClientAuthInfoWriter) (*ReplacePoolsByGroupIDOK, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
  AddNewGroup groups

  Create new group
*/
func (a *Client) AddNewGroup(params *AddNewGroupParams, authInfo runtime.ClientAuthInfoWriter) (*AddNewGroupOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewAddNewGroupParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "addNewGroup",
		Method:             "POST",
		PathPattern:        "/groups",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &AddNewGroupReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*AddNewGroupOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for addNewGroup: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  AddPoolsByGroupID groups

  Add a list of pool_ids in the group (keep existing). Return new complete list.
*/
func (a *Client) AddPoolsByGroupID(params *AddPoolsByGroupIDParams, authInfo runtime.ClientAuthInfoWriter) (*AddPoolsByGroupIDOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewAddPoolsByGroupIDParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "addPoolsByGroupID",
		Method:             "POST",
		PathPattern:        "/groups/{group_id}/pools",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &AddPoolsByGroupIDReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*AddPoolsByGroupIDOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for addPoolsByGroupID: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  DeleteGroup groups

  Delete this group, but not the pools associated with it.
*/
func (a *Client) DeleteGroup(params *DeleteGroupParams, authInfo runtime.ClientAuthInfoWriter) error {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDeleteGroupParams()
	}

	_, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "deleteGroup",
		Method:             "DELETE",
		PathPattern:        "/groups/{group_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &DeleteGroupReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return err
	}
	return nil
}

/*
  DeletePoolsByGroupID groups

  Delete one or more pool_ids in the group. Return new complete list.
*/
func (a *Client) DeletePoolsByGroupID(params *DeletePoolsByGroupIDParams, authInfo runtime.ClientAuthInfoWriter) (*DeletePoolsByGroupIDOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDeletePoolsByGroupIDParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "deletePoolsByGroupID",
		Method:             "DELETE",
		PathPattern:        "/groups/{group_id}/pools",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &DeletePoolsByGroupIDReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*DeletePoolsByGroupIDOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for deletePoolsByGroupID: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  GetGroupDescriptionByID groups

  Gets a description of a group
*/
func (a *Client) GetGroupDescriptionByID(params *GetGroupDescriptionByIDParams, authInfo runtime.ClientAuthInfoWriter) (*GetGroupDescriptionByIDOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetGroupDescriptionByIDParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "getGroupDescriptionByID",
		Method:             "GET",
		PathPattern:        "/groups/{group_id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetGroupDescriptionByIDReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetGroupDescriptionByIDOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for getGroupDescriptionByID: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  GetGroupIDByName groups

  Gets group id for a given group name
*/
func (a *Client) GetGroupIDByName(params *GetGroupIDByNameParams, authInfo runtime.ClientAuthInfoWriter) (*GetGroupIDByNameOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetGroupIDByNameParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "getGroupIDByName",
		Method:             "GET",
		PathPattern:        "/groups",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetGroupIDByNameReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetGroupIDByNameOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for getGroupIDByName: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  GetPoolsByGroupID groups

  Gets a list of pool_ids in the group
*/
func (a *Client) GetPoolsByGroupID(params *GetPoolsByGroupIDParams, authInfo runtime.ClientAuthInfoWriter) (*GetPoolsByGroupIDOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewGetPoolsByGroupIDParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "getPoolsByGroupID",
		Method:             "GET",
		PathPattern:        "/groups/{group_id}/pools",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &GetPoolsByGroupIDReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*GetPoolsByGroupIDOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for getPoolsByGroupID: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  ReplacePoolsByGroupID groups

  Update list of pool_ids in the group (replace existing). Return new complete list.
*/
func (a *Client) ReplacePoolsByGroupID(params *ReplacePoolsByGroupIDParams, authInfo runtime.ClientAuthInfoWriter) (*ReplacePoolsByGroupIDOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewReplacePoolsByGroupIDParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "replacePoolsByGroupID",
		Method:             "PUT",
		PathPattern:        "/groups/{group_id}/pools",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"https"},
		Params:             params,
		Reader:             &ReplacePoolsByGroupIDReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	success, ok := result.(*ReplacePoolsByGroupIDOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for replacePoolsByGroupID: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
