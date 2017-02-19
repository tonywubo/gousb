// Copyright 2013 Google Inc.  All rights reserved.
// Copyright 2016 the gousb Authors.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package usb

import (
	"fmt"
	"log"
	"time"
)

type Endpoint interface {
	Read(b []byte) (int, error)
	Write(b []byte) (int, error)
	Interface() InterfaceSetup
	Info() EndpointInfo
}

type transferIntf interface {
	submit() error
	wait() (int, error)
	free() error
}

type endpoint struct {
	*Device
	InterfaceSetup
	EndpointInfo
	newUSBTransfer func([]byte, time.Duration) (transferIntf, error)
}

func (e *endpoint) Read(buf []byte) (int, error) {
	if EndpointDirection(e.Address)&ENDPOINT_DIR_MASK != ENDPOINT_DIR_IN {
		return 0, fmt.Errorf("usb: read: not an IN endpoint")
	}

	return e.transfer(buf, e.ReadTimeout)
}

func (e *endpoint) Write(buf []byte) (int, error) {
	if EndpointDirection(e.Address)&ENDPOINT_DIR_MASK != ENDPOINT_DIR_OUT {
		return 0, fmt.Errorf("usb: write: not an OUT endpoint")
	}

	return e.transfer(buf, e.WriteTimeout)
}

func (e *endpoint) Interface() InterfaceSetup { return e.InterfaceSetup }
func (e *endpoint) Info() EndpointInfo        { return e.EndpointInfo }

func (e *endpoint) newRealUSBTransfer(buf []byte, timeout time.Duration) (transferIntf, error) {
	return newUSBTransfer((*deviceHandle)(e.Device.handle), e.EndpointInfo, buf, timeout)
}

func (e *endpoint) transfer(buf []byte, timeout time.Duration) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}

	tt := e.TransferType()
	t, err := e.newUSBTransfer(buf, timeout)
	if err != nil {
		return 0, err
	}
	defer t.free()

	if err := t.submit(); err != nil {
		log.Printf("bulk: %s failed to submit: %s", tt, err)
		return 0, err
	}

	n, err := t.wait()
	if err != nil {
		log.Printf("bulk: %s failed: %s", tt, err)
		return 0, err
	}
	return n, err
}

func newEndpoint(d *Device) *endpoint {
	ep := &endpoint{
		Device: d,
	}
	ep.newUSBTransfer = ep.newRealUSBTransfer
	return ep
}
