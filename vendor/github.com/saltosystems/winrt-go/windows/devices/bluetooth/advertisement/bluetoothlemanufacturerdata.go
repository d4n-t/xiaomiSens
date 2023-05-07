// Code generated by winrt-go-gen. DO NOT EDIT.

//go:build windows

//nolint
package advertisement

import (
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/saltosystems/winrt-go/windows/storage/streams"
)

const SignatureBluetoothLEManufacturerData string = "rc(Windows.Devices.Bluetooth.Advertisement.BluetoothLEManufacturerData;{912dba18-6963-4533-b061-4694dafb34e5})"

type BluetoothLEManufacturerData struct {
	ole.IUnknown
}

func NewBluetoothLEManufacturerData() (*BluetoothLEManufacturerData, error) {
	inspectable, err := ole.RoActivateInstance("Windows.Devices.Bluetooth.Advertisement.BluetoothLEManufacturerData")
	if err != nil {
		return nil, err
	}
	return (*BluetoothLEManufacturerData)(unsafe.Pointer(inspectable)), nil
}

func (impl *BluetoothLEManufacturerData) GetCompanyId() (uint16, error) {
	itf := impl.MustQueryInterface(ole.NewGUID(GUIDiBluetoothLEManufacturerData))
	defer itf.Release()
	v := (*iBluetoothLEManufacturerData)(unsafe.Pointer(itf))
	return v.GetCompanyId()
}

func (impl *BluetoothLEManufacturerData) SetCompanyId(value uint16) error {
	itf := impl.MustQueryInterface(ole.NewGUID(GUIDiBluetoothLEManufacturerData))
	defer itf.Release()
	v := (*iBluetoothLEManufacturerData)(unsafe.Pointer(itf))
	return v.SetCompanyId(value)
}

func (impl *BluetoothLEManufacturerData) GetData() (*streams.IBuffer, error) {
	itf := impl.MustQueryInterface(ole.NewGUID(GUIDiBluetoothLEManufacturerData))
	defer itf.Release()
	v := (*iBluetoothLEManufacturerData)(unsafe.Pointer(itf))
	return v.GetData()
}

func (impl *BluetoothLEManufacturerData) SetData(value *streams.IBuffer) error {
	itf := impl.MustQueryInterface(ole.NewGUID(GUIDiBluetoothLEManufacturerData))
	defer itf.Release()
	v := (*iBluetoothLEManufacturerData)(unsafe.Pointer(itf))
	return v.SetData(value)
}

const GUIDiBluetoothLEManufacturerData string = "912dba18-6963-4533-b061-4694dafb34e5"
const SignatureiBluetoothLEManufacturerData string = "{912dba18-6963-4533-b061-4694dafb34e5}"

type iBluetoothLEManufacturerData struct {
	ole.IInspectable
}

type iBluetoothLEManufacturerDataVtbl struct {
	ole.IInspectableVtbl

	GetCompanyId uintptr
	SetCompanyId uintptr
	GetData      uintptr
	SetData      uintptr
}

func (v *iBluetoothLEManufacturerData) VTable() *iBluetoothLEManufacturerDataVtbl {
	return (*iBluetoothLEManufacturerDataVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *iBluetoothLEManufacturerData) GetCompanyId() (uint16, error) {
	var out uint16
	hr, _, _ := syscall.SyscallN(
		v.VTable().GetCompanyId,
		uintptr(unsafe.Pointer(v)),    // this
		uintptr(unsafe.Pointer(&out)), // out uint16
	)

	if hr != 0 {
		return 0, ole.NewError(hr)
	}

	return out, nil
}

func (v *iBluetoothLEManufacturerData) SetCompanyId(value uint16) error {
	hr, _, _ := syscall.SyscallN(
		v.VTable().SetCompanyId,
		uintptr(unsafe.Pointer(v)), // this
		uintptr(value),             // in uint16
	)

	if hr != 0 {
		return ole.NewError(hr)
	}

	return nil
}

func (v *iBluetoothLEManufacturerData) GetData() (*streams.IBuffer, error) {
	var out *streams.IBuffer
	hr, _, _ := syscall.SyscallN(
		v.VTable().GetData,
		uintptr(unsafe.Pointer(v)),    // this
		uintptr(unsafe.Pointer(&out)), // out streams.IBuffer
	)

	if hr != 0 {
		return nil, ole.NewError(hr)
	}

	return out, nil
}

func (v *iBluetoothLEManufacturerData) SetData(value *streams.IBuffer) error {
	hr, _, _ := syscall.SyscallN(
		v.VTable().SetData,
		uintptr(unsafe.Pointer(v)),     // this
		uintptr(unsafe.Pointer(value)), // in streams.IBuffer
	)

	if hr != 0 {
		return ole.NewError(hr)
	}

	return nil
}

const GUIDiBluetoothLEManufacturerDataFactory string = "c09b39f8-319a-441e-8de5-66a81e877a6c"
const SignatureiBluetoothLEManufacturerDataFactory string = "{c09b39f8-319a-441e-8de5-66a81e877a6c}"

type iBluetoothLEManufacturerDataFactory struct {
	ole.IInspectable
}

type iBluetoothLEManufacturerDataFactoryVtbl struct {
	ole.IInspectableVtbl

	Create uintptr
}

func (v *iBluetoothLEManufacturerDataFactory) VTable() *iBluetoothLEManufacturerDataFactoryVtbl {
	return (*iBluetoothLEManufacturerDataFactoryVtbl)(unsafe.Pointer(v.RawVTable))
}

func Create(companyId uint16, data *streams.IBuffer) (*BluetoothLEManufacturerData, error) {
	inspectable, err := ole.RoGetActivationFactory("Windows.Devices.Bluetooth.Advertisement.BluetoothLEManufacturerData", ole.NewGUID(GUIDiBluetoothLEManufacturerDataFactory))
	if err != nil {
		return nil, err
	}
	v := (*iBluetoothLEManufacturerDataFactory)(unsafe.Pointer(inspectable))

	var out *BluetoothLEManufacturerData
	hr, _, _ := syscall.SyscallN(
		v.VTable().Create,
		0,                             // this is a static func, so there's no this
		uintptr(companyId),            // in uint16
		uintptr(unsafe.Pointer(data)), // in streams.IBuffer
		uintptr(unsafe.Pointer(&out)), // out BluetoothLEManufacturerData
	)

	if hr != 0 {
		return nil, ole.NewError(hr)
	}

	return out, nil
}