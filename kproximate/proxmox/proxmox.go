package proxmox

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/Telmate/proxmox-api-go/proxmox"
	"github.com/mitchellh/mapstructure"
)

var exitStatusSuccess = regexp.MustCompile(`^(OK|WARNINGS)`)
var userRequiresTokenRegex = regexp.MustCompile("[a-z0-9]+@[a-z0-9]+![a-z0-9]+")

type HostInformation struct {
	Id     string  `json:"id"`
	Node   string  `json:"node"`
	Cpu    float64 `json:"cpu"`
	Mem    int64   `json:"mem"`
	Maxmem int64   `json:"maxmem"`
	Status string  `json:"status"`
}

type vmList struct {
	Data []VmInformation
}

type VmInformation struct {
	VmID    int     `json:"vmid"`
	Name    string  `json:"name"`
	Cpu     float64 `json:"cpu"`
	CpuType string  `json:"cputype"`
	Status  string  `json:"status"`
	MaxMem  int64   `json:"maxmem"`
	Mem     int64   `json:"mem"`
	MaxDisk int64   `json:"maxdisk"`
	NetIn   int64   `json:"netin"`
	NetOut  int64   `json:"netout"`
	Node    string  `json:"node"`
	Uptime  int     `json:"uptime"`
}

type QemuExecResponse struct {
	Pid int `json:"pid"`
}

type QemuExecStatus struct {
	Exited   int    `mapstructure:"exited"`
	ExitCode int    `mapstructure:"exitcode"`
	ErrData  string `mapstructure:"err-data"`
	OutData  string `mapstructure:"out-data"`
}

type Proxmox interface {
	GetClusterStats(ctx context.Context) ([]HostInformation, error)
	GetRunningKpNodes(ctx context.Context, kpNodeNameRegex regexp.Regexp) ([]VmInformation, error)
	GetAllKpNodes(ctx context.Context, kpNodeNameRegex regexp.Regexp) ([]VmInformation, error)
	GetKpNode(ctx context.Context, name string, kpNodeNameRegex regexp.Regexp) (VmInformation, error)
	GetKpNodeTemplateRef(ctx context.Context, kpNodeTemplateName string, localTemplateStorage bool, cloneTargetNode string) (*proxmox.VmRef, error)
	NewKpNode(ctx context.Context, newKpNodeName string, targetNode string, kpNodeParams map[string]interface{}, usingLocalStorage bool, kpNodeTemplateName string, kpJoinCommand string) error
	DeleteKpNode(ctx context.Context, name string, kpnodeName regexp.Regexp) error
	QemuExecJoin(ctx context.Context, nodeName string, joinCommand string) (int, error)
	GetQemuExecJoinStatus(ctx context.Context, nodeName string, pid int) (QemuExecStatus, error)
	CheckNodeReady(ctx context.Context, nodeName string) error
}

type ProxmoxClientInterface interface {
	CloneQemuVm(ctx context.Context, vmr *proxmox.VmRef, vmParams map[string]interface{}) (exitStatus string, err error)
	DeleteVm(ctx context.Context, vmr *proxmox.VmRef) (exitStatus string, err error)
	GetExecStatus(ctx context.Context, vmr *proxmox.VmRef, pid string) (status map[string]interface{}, err error)
	GetNextID(ctx context.Context, currentID *proxmox.GuestID) (nextID proxmox.GuestID, err error)
	GetResourceList(ctx context.Context, resourceType string) (list []interface{}, err error)
	GetVmList(ctx context.Context) (map[string]interface{}, error)
	GetVmRefByName(ctx context.Context, vmName proxmox.GuestName) (vmr *proxmox.VmRef, err error)
	GetVmRefsByName(ctx context.Context, vmName proxmox.GuestName) (vmrs []*proxmox.VmRef, err error)
	QemuAgentExec(ctx context.Context, vmr *proxmox.VmRef, params map[string]interface{}) (result map[string]interface{}, err error)
	QemuAgentPing(ctx context.Context, vmr *proxmox.VmRef) (pingRes map[string]interface{}, err error)
	SetVmConfig(vmr *proxmox.VmRef, params map[string]interface{}) (exitStatus interface{}, err error)
	StartVm(ctx context.Context, vmr *proxmox.VmRef) (exitStatus string, err error)
	StopVm(ctx context.Context, vmr *proxmox.VmRef) (exitStatus string, err error)
}

type ProxmoxClient struct {
	client ProxmoxClientInterface
}

func userRequiresAPIToken(pmUser string) bool {
	return userRequiresTokenRegex.MatchString(pmUser)
}

func NewProxmoxClient(ctx context.Context, pm_url string, allowInsecure bool, pmUser string, pmToken string, pmPassword string, debug bool) (ProxmoxClient, error) {
	tlsconf := &tls.Config{InsecureSkipVerify: allowInsecure}
	newClient, err := proxmox.NewClient(pm_url, nil, "", tlsconf, "", 300)
	if err != nil {
		return ProxmoxClient{}, err
	}

	if userRequiresAPIToken(pmUser) {
		newClient.SetAPIToken(pmUser, pmToken)
	} else {
		err = newClient.Login(ctx, pmUser, pmPassword, "")
		if err != nil {
			return ProxmoxClient{}, err
		}
	}

	proxmox.Debug = &debug

	proxmox := ProxmoxClient{
		client: newClient,
	}

	return proxmox, nil
}

func (p *ProxmoxClient) GetClusterStats(ctx context.Context) ([]HostInformation, error) {
	hostList, err := p.client.GetResourceList(ctx, "node")
	if err != nil {
		return nil, err
	}

	var pHosts []HostInformation

	err = mapstructure.Decode(hostList, &pHosts)
	if err != nil {
		return nil, err
	}

	return pHosts, nil
}

func (p *ProxmoxClient) GetAllKpNodes(ctx context.Context, kpNodeNameRegex regexp.Regexp) ([]VmInformation, error) {
	result, err := p.client.GetVmList(ctx)
	if err != nil {
		return nil, err
	}

	var vmlist vmList

	err = mapstructure.Decode(result, &vmlist)
	if err != nil {
		return nil, err
	}

	var kpNodes []VmInformation

	for _, vm := range vmlist.Data {
		if kpNodeNameRegex.MatchString(vm.Name) {
			kpNodes = append(kpNodes, vm)
		}
	}

	return kpNodes, err
}

func (p *ProxmoxClient) GetRunningKpNodes(ctx context.Context, kpNodeNameRegex regexp.Regexp) ([]VmInformation, error) {
	kpNodes, err := p.GetAllKpNodes(ctx, kpNodeNameRegex)
	if err != nil {
		return nil, err
	}

	var runningKpNodes []VmInformation

	for _, vm := range kpNodes {
		if vm.Status == "running" {
			runningKpNodes = append(runningKpNodes, vm)
		}
	}

	return runningKpNodes, nil
}

func (p *ProxmoxClient) GetKpNode(ctx context.Context, kpNodeName string, kpNodeNameRegex regexp.Regexp) (VmInformation, error) {
	kpNodes, err := p.GetAllKpNodes(ctx, kpNodeNameRegex)
	if err != nil {
		return VmInformation{}, err
	}

	for _, vm := range kpNodes {
		if vm.Name == kpNodeName {
			return vm, err
		}
	}

	return VmInformation{}, err
}

func (p *ProxmoxClient) GetKpNodeTemplateRef(ctx context.Context, kpNodeTemplateName string, localTemplateStorage bool, cloneTargetNode string) (*proxmox.VmRef, error) {
	vmRefs, err := p.client.GetVmRefsByName(ctx, proxmox.GuestName(kpNodeTemplateName))
	if err != nil {
		return nil, err
	}

	if localTemplateStorage {
		for _, vmRef := range vmRefs {
			if vmRef.Node().String() == cloneTargetNode {
				return vmRef, nil
			}
		}
	} else {
		return vmRefs[0], nil
	}

	return nil, fmt.Errorf("could not find template: %s", kpNodeTemplateName)
}

func getNextID(ctx context.Context, proxmoxClient *ProxmoxClient, currentID *proxmox.GuestID) (proxmox.GuestID, error) {
	nextID, err := proxmoxClient.client.GetNextID(ctx, currentID)
	if err != nil {
		if err.Error() == "400 Parameter verification failed." {
			*currentID++
			return getNextID(ctx, proxmoxClient, currentID)
		}

		return 0, err
	}

	return nextID, nil
}

func (p *ProxmoxClient) NewKpNode(
	ctx context.Context,
	newKpNodeName string,
	targetNode string,
	kpNodeParams map[string]interface{},
	localTemplateStorage bool,
	kpNodeTemplateName string,
	kpJoinCommand string,
) error {
	kpNodeTemplate, err := p.GetKpNodeTemplateRef(ctx, kpNodeTemplateName, localTemplateStorage, targetNode)
	if err != nil {
		return err
	}

	currentID := kpNodeTemplate.VmId()
	nextID, err := getNextID(ctx, p, &currentID)
	if err != nil {
		return err
	}

	cloneParams := map[string]interface{}{
		"name":   newKpNodeName,
		"newid":  nextID,
		"target": targetNode,
		"vmid":   kpNodeTemplate.VmId(),
	}

	_, err = p.client.CloneQemuVm(ctx, kpNodeTemplate, cloneParams)
	if err != nil {
		return err
	}

	for {
		newVmRef, err := p.client.GetVmRefByName(ctx, proxmox.GuestName(newKpNodeName))
		if err != nil {
			time.Sleep(time.Second * 1)
			continue
		}

		_, err = p.client.SetVmConfig(newVmRef, kpNodeParams)
		if err != nil {
			return err
		}

		_, err = p.client.StartVm(ctx, newVmRef)
		if err != nil {
			return err
		}
		break
	}

	return nil
}

func (p *ProxmoxClient) CheckNodeReady(ctx context.Context, nodeName string) error {
	vmRef, err := p.client.GetVmRefByName(ctx, proxmox.GuestName(nodeName))
	if err != nil {
		return err
	}

	_, pingErr := p.client.QemuAgentPing(ctx, vmRef)

	for pingErr != nil {
		_, pingErr = p.client.QemuAgentPing(ctx, vmRef)
		time.Sleep(time.Second * 1)
	}

	return nil
}

func (p *ProxmoxClient) QemuExecJoin(ctx context.Context, kpNodeName string, joinCommand string) (int, error) {
	vmRef, err := p.client.GetVmRefByName(ctx, proxmox.GuestName(kpNodeName))
	if err != nil {
		return 0, err
	}

	params := map[string]interface{}{
		"command": []string{"bash", "-c", joinCommand},
	}

	result, err := p.client.QemuAgentExec(ctx, vmRef, params)
	if err != nil {
		return 0, err
	}

	var response QemuExecResponse

	err = mapstructure.Decode(result, &response)
	if err != nil {
		return 0, err
	}

	return response.Pid, nil
}

func (p *ProxmoxClient) GetQemuExecJoinStatus(ctx context.Context, kpNodeName string, pid int) (QemuExecStatus, error) {
	vmRef, err := p.client.GetVmRefByName(ctx, proxmox.GuestName(kpNodeName))
	if err != nil {
		return QemuExecStatus{}, err
	}

	execStatus, err := p.client.GetExecStatus(ctx, vmRef, fmt.Sprintf("%d", pid))
	if err != nil {
		return QemuExecStatus{}, err
	}

	var status QemuExecStatus

	err = mapstructure.Decode(execStatus, &status)
	if err != nil {
		return QemuExecStatus{}, err
	}

	return status, nil
}

func (p *ProxmoxClient) DeleteKpNode(ctx context.Context, name string, kpNodeName regexp.Regexp) error {
	kpNode, err := p.GetKpNode(ctx, name, kpNodeName)
	if err != nil {
		return err
	}

	vmRef, err := p.client.GetVmRefByName(ctx, proxmox.GuestName(kpNode.Name))
	if err != nil {
		return err
	}

	exitStatus, err := p.client.StopVm(ctx, vmRef)
	if err != nil {
		return err
	}

	if !exitStatusSuccess.MatchString(exitStatus) {
		err = errors.New(exitStatus)
		return err
	}

	exitStatus, err = p.client.DeleteVm(ctx, vmRef)
	if err != nil {
		return err
	}

	if !exitStatusSuccess.MatchString(exitStatus) {
		err = errors.New(exitStatus)
		return err
	}

	return nil
}
