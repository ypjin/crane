// Copyright 2015 go-dockerclient authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package docker

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/net/context"
)

// ErrNetworkAlreadyExists is the error returned by CreateNetwork when the
// network already exists.
var ErrNetworkAlreadyExists = errors.New("network already exists")

// Network represents a network.
//
// See https://goo.gl/6GugX3 for more details.
type Network struct {
	Name       string
	ID         string `json:"Id"`
	Scope      string
	Driver     string
	IPAM       IPAMOptions
	Containers map[string]Endpoint
	Options    map[string]string
	Internal   bool
	EnableIPv6 bool `json:"EnableIPv6"`
	Labels     map[string]string
}

// Endpoint contains network resources allocated and used for a container in a network
//
// See https://goo.gl/6GugX3 for more details.
type Endpoint struct {
	Name        string
	ID          string `json:"EndpointID"`
	MacAddress  string
	IPv4Address string
	IPv6Address string
}

// ListNetworks returns all networks.
//
// See https://goo.gl/6GugX3 for more details.
func (c *Client) ListNetworks() ([]Network, error) {
	resp, err := c.do("GET", "/networks", doOptions{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var networks []Network
	if err := json.NewDecoder(resp.Body).Decode(&networks); err != nil {
		return nil, err
	}
	return networks, nil
}

// NetworkFilterOpts is an aggregation of key=value that Docker
// uses to filter networks
type NetworkFilterOpts map[string]map[string]bool

// FilteredListNetworks returns all networks with the filters applied
//
// See goo.gl/zd2mx4 for more details.
func (c *Client) FilteredListNetworks(opts NetworkFilterOpts) ([]Network, error) {
	params, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	path := "/networks?filters=" + string(params)
	resp, err := c.do("GET", path, doOptions{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var networks []Network
	if err := json.NewDecoder(resp.Body).Decode(&networks); err != nil {
		return nil, err
	}
	return networks, nil
}

// NetworkInfo returns information about a network by its ID.
//
// See https://goo.gl/6GugX3 for more details.
func (c *Client) NetworkInfo(id string) (*Network, error) {
	path := "/networks/" + id
	resp, err := c.do("GET", path, doOptions{})
	if err != nil {
		if e, ok := err.(*Error); ok && e.Status == http.StatusNotFound {
			return nil, &NoSuchNetwork{ID: id}
		}
		return nil, err
	}
	defer resp.Body.Close()
	var network Network
	if err := json.NewDecoder(resp.Body).Decode(&network); err != nil {
		return nil, err
	}
	return &network, nil
}

// CreateNetworkOptions specify parameters to the CreateNetwork function and
// (for now) is the expected body of the "create network" http request message
//
// See https://goo.gl/6GugX3 for more details.
type CreateNetworkOptions struct {
	Name           string                 `json:"Name" yaml:"Name"`
	CheckDuplicate bool                   `json:"CheckDuplicate" yaml:"CheckDuplicate"`
	Driver         string                 `json:"Driver" yaml:"Driver"`
	IPAM           IPAMOptions            `json:"IPAM" yaml:"IPAM"`
	Options        map[string]interface{} `json:"Options" yaml:"Options"`
	Labels         map[string]string      `json:"Labels" yaml:"Labels"`
	Internal       bool                   `json:"Internal" yaml:"Internal"`
	EnableIPv6     bool                   `json:"EnableIPv6" yaml:"EnableIPv6"`
	Context        context.Context        `json:"-"`
}

// IPAMOptions controls IP Address Management when creating a network
//
// See https://goo.gl/T8kRVH for more details.
type IPAMOptions struct {
	Driver string       `json:"Driver" yaml:"Driver"`
	Config []IPAMConfig `json:"Config" yaml:"Config"`
}

// IPAMConfig represents IPAM configurations
//
// See https://goo.gl/T8kRVH for more details.
type IPAMConfig struct {
	Subnet     string            `json:",omitempty"`
	IPRange    string            `json:",omitempty"`
	Gateway    string            `json:",omitempty"`
	AuxAddress map[string]string `json:"AuxiliaryAddresses,omitempty"`
}

// CreateNetwork creates a new network, returning the network instance,
// or an error in case of failure.
//
// See https://goo.gl/6GugX3 for more details.
func (c *Client) CreateNetwork(opts CreateNetworkOptions) (*Network, error) {
	resp, err := c.do(
		"POST",
		"/networks/create",
		doOptions{
			data:    opts,
			context: opts.Context,
		},
	)
	if err != nil {
		if e, ok := err.(*Error); ok && e.Status == http.StatusConflict {
			return nil, ErrNetworkAlreadyExists
		}
		return nil, err
	}
	defer resp.Body.Close()

	type createNetworkResponse struct {
		ID string
	}
	var (
		network Network
		cnr     createNetworkResponse
	)
	if err := json.NewDecoder(resp.Body).Decode(&cnr); err != nil {
		return nil, err
	}

	network.Name = opts.Name
	network.ID = cnr.ID
	network.Driver = opts.Driver

	return &network, nil
}

// RemoveNetwork removes a network or returns an error in case of failure.
//
// See https://goo.gl/6GugX3 for more details.
func (c *Client) RemoveNetwork(id string) error {
	resp, err := c.do("DELETE", "/networks/"+id, doOptions{})
	if err != nil {
		if e, ok := err.(*Error); ok && e.Status == http.StatusNotFound {
			return &NoSuchNetwork{ID: id}
		}
		return err
	}
	resp.Body.Close()
	return nil
}

// NetworkConnectionOptions specify parameters to the ConnectNetwork and
// DisconnectNetwork function.
//
// See https://goo.gl/RV7BJU for more details.
type NetworkConnectionOptions struct {
	Container string

	// EndpointConfig is only applicable to the ConnectNetwork call
	EndpointConfig *EndpointConfig `json:"EndpointConfig,omitempty"`

	// Force is only applicable to the DisconnectNetwork call
	Force bool

	Context context.Context `json:"-"`
}

// EndpointConfig stores network endpoint details
//
// See https://goo.gl/RV7BJU for more details.
type EndpointConfig struct {
	IPAMConfig          *EndpointIPAMConfig `json:"IPAMConfig,omitempty" yaml:"IPAMConfig,omitempty"`
	Links               []string            `json:"Links,omitempty" yaml:"Links,omitempty"`
	Aliases             []string            `json:"Aliases,omitempty" yaml:"Aliases,omitempty"`
	NetworkID           string              `json:"NetworkID,omitempty" yaml:"NetworkID,omitempty"`
	EndpointID          string              `json:"EndpointID,omitempty" yaml:"EndpointID,omitempty"`
	Gateway             string              `json:"Gateway,omitempty" yaml:"Gateway,omitempty"`
	IPAddress           string              `json:"IPAddress,omitempty" yaml:"IPAddress,omitempty"`
	IPPrefixLen         int                 `json:"IPPrefixLen,omitempty" yaml:"IPPrefixLen,omitempty"`
	IPv6Gateway         string              `json:"IPv6Gateway,omitempty" yaml:"IPv6Gateway,omitempty"`
	GlobalIPv6Address   string              `json:"GlobalIPv6Address,omitempty" yaml:"GlobalIPv6Address,omitempty"`
	GlobalIPv6PrefixLen int                 `json:"GlobalIPv6PrefixLen,omitempty" yaml:"GlobalIPv6PrefixLen,omitempty"`
	MacAddress          string              `json:"MacAddress,omitempty" yaml:"MacAddress,omitempty"`
}

// EndpointIPAMConfig represents IPAM configurations for an
// endpoint
//
// See https://goo.gl/RV7BJU for more details.
type EndpointIPAMConfig struct {
	IPv4Address string `json:",omitempty"`
	IPv6Address string `json:",omitempty"`
}

// ConnectNetwork adds a container to a network or returns an error in case of
// failure.
//
// See https://goo.gl/6GugX3 for more details.
func (c *Client) ConnectNetwork(id string, opts NetworkConnectionOptions) error {
	resp, err := c.do("POST", "/networks/"+id+"/connect", doOptions{
		data:    opts,
		context: opts.Context,
	})
	if err != nil {
		if e, ok := err.(*Error); ok && e.Status == http.StatusNotFound {
			return &NoSuchNetworkOrContainer{NetworkID: id, ContainerID: opts.Container}
		}
		return err
	}
	resp.Body.Close()
	return nil
}

// DisconnectNetwork removes a container from a network or returns an error in
// case of failure.
//
// See https://goo.gl/6GugX3 for more details.
func (c *Client) DisconnectNetwork(id string, opts NetworkConnectionOptions) error {
	resp, err := c.do("POST", "/networks/"+id+"/disconnect", doOptions{data: opts})
	if err != nil {
		if e, ok := err.(*Error); ok && e.Status == http.StatusNotFound {
			return &NoSuchNetworkOrContainer{NetworkID: id, ContainerID: opts.Container}
		}
		return err
	}
	resp.Body.Close()
	return nil
}

// NoSuchNetwork is the error returned when a given network does not exist.
type NoSuchNetwork struct {
	ID string
}

func (err *NoSuchNetwork) Error() string {
	return fmt.Sprintf("No such network: %s", err.ID)
}

// NoSuchNetworkOrContainer is the error returned when a given network or
// container does not exist.
type NoSuchNetworkOrContainer struct {
	NetworkID   string
	ContainerID string
}

func (err *NoSuchNetworkOrContainer) Error() string {
	return fmt.Sprintf("No such network (%s) or container (%s)", err.NetworkID, err.ContainerID)
}
