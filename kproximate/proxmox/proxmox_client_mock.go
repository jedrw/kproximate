package proxmox

import (
	"context"

	"github.com/Telmate/proxmox-api-go/proxmox"
)

type ProxmoxClientMock struct {
	ExecStatus            map[string]interface{}
	NextID                int
	ResourceList          []interface{}
	VmList                map[string]interface{}
	VmRefByName           map[proxmox.GuestName]*proxmox.VmRef
	VmRefsByName          map[proxmox.GuestName][]*proxmox.VmRef
	QemuExecResponse      map[string]interface{}
	QemuAgentPingResponse map[string]interface{}
}

func (m *ProxmoxClientMock) CloneQemuVm(ctx context.Context, vmr *proxmox.VmRef, vmParams map[string]interface{}) (exitStatus string, err error) {
	return "OK", nil
}

func (m *ProxmoxClientMock) DeleteVm(ctx context.Context, vmr *proxmox.VmRef) (exitStatus string, err error) {
	return "OK", nil
}

func (m *ProxmoxClientMock) GetExecStatus(ctx context.Context, vmr *proxmox.VmRef, pid string) (status map[string]interface{}, err error) {
	return m.ExecStatus, nil
}

func (m *ProxmoxClientMock) GetNextID(ctx context.Context, currentID *proxmox.GuestID) (nextID proxmox.GuestID, err error) {
	m.NextID++
	return proxmox.GuestID(m.NextID), nil
}

func (m *ProxmoxClientMock) GetResourceList(ctx context.Context, resourceType string) (list []interface{}, err error) {
	return m.ResourceList, nil
}

func (m *ProxmoxClientMock) GetVmList(ctx context.Context) (map[string]interface{}, error) {
	return m.VmList, nil
}

func (m *ProxmoxClientMock) GetVmRefByName(ctx context.Context, vmName proxmox.GuestName) (vmr *proxmox.VmRef, err error) {
	return m.VmRefByName[vmName], nil
}

func (m *ProxmoxClientMock) GetVmRefsByName(ctx context.Context, vmName proxmox.GuestName) (vmrs []*proxmox.VmRef, err error) {
	return m.VmRefsByName[vmName], nil

}

func (m *ProxmoxClientMock) QemuAgentExec(ctx context.Context, vmr *proxmox.VmRef, params map[string]interface{}) (result map[string]interface{}, err error) {
	return m.QemuExecResponse, nil
}

func (m *ProxmoxClientMock) QemuAgentPing(ctx context.Context, vmr *proxmox.VmRef) (pingRes map[string]interface{}, err error) {
	return m.QemuAgentPingResponse, nil
}

func (m *ProxmoxClientMock) SetVmConfig(vmr *proxmox.VmRef, params map[string]interface{}) (exitStatus interface{}, err error) {
	return "OK", nil
}

func (m *ProxmoxClientMock) StartVm(ctx context.Context, vmr *proxmox.VmRef) (exitStatus string, err error) {
	return "OK", nil
}

func (m *ProxmoxClientMock) StopVm(ctx context.Context, vmr *proxmox.VmRef) (exitStatus string, err error) {
	return "OK", nil
}
