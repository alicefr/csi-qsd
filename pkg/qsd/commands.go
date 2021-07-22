package qsd

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/digitalocean/go-qemu/qmp"
)

type VolumeManager struct {
	Monitor *QMPMonitor
}

func NewVolumeManager(socket string) (*VolumeManager, error) {
	if socket == "" {
		return nil, fmt.Errorf("The socket cannot be empty")
	}
	q, err := CreateNewUnixMonitor(socket)
	if err != nil {
		return nil, err
	}
	return &VolumeManager{Monitor: q}, nil
}

func (v *VolumeManager) Disconnect() {
	v.Monitor.Disconnect()
}

type statusResult struct {
	ID     string `json:"id"`
	Return struct {
		Running    bool   `json:"running"`
		Singlestep bool   `json:"singlestep"`
		Status     string `json:"status"`
	} `json:"return"`
}

type QMPMonitor struct {
	monitor qmp.Monitor
}

func CreateNewUnixMonitor(socket string) (*QMPMonitor, error) {
	m, err := qmp.NewSocketMonitor("unix", socket, 2*time.Second)
	if err != nil {
		return &QMPMonitor{}, fmt.Errorf("Fail in creating qmp connection: %v", err)
	}
	err = m.Connect()
	if err != nil {
		return &QMPMonitor{}, err
	}
	return &QMPMonitor{monitor: m}, nil
}

func (q *QMPMonitor) Disconnect() {
	q.monitor.Disconnect()
}

func (q *QMPMonitor) ExecuteCommand(qmpCmd string) error {
	cmd := []byte(qmpCmd)
	raw, err := q.monitor.Run(cmd)
	if err != nil {
		return fmt.Errorf("failed running qmp command %s: %v", qmpCmd, err)
	}
	fmt.Printf("Executed command %s result: %s \n", qmpCmd, string(raw))
	var result statusResult
	err = json.Unmarshal(raw, &result)
	if err != nil {
		return fmt.Errorf("failed parsing result %v", err)
	}

	fmt.Println(result.Return.Status)
	return nil
}

const (
	GB = 1024 * 1024 * 1024
	MB = 1024 * 1024
	KB = 1024
)

func parseSizeToByteString(size string) (string, error) {
	s := strings.ReplaceAll(size, " ", "")
	unit := s[len(s)-2 : len(s)]
	regDigit, err := regexp.Compile("[^0-9]+")
	if err != nil {
		return "", err
	}
	regLetter, err := regexp.Compile("[^a-zA-Z]+")
	if err != nil {
		return "", err
	}
	u := regLetter.ReplaceAllString(unit, "")
	var q int
	q, err = strconv.Atoi(regDigit.ReplaceAllString(s, ""))
	if err != nil {
		return "", err
	}
	switch u {
	case "M", "MB":
		return strconv.Itoa(q * MB), nil
	case "G", "GB":
		return strconv.Itoa(q * GB), nil
	}
	return "", fmt.Errorf("Quantity %s not supported", u)
}

func (v *VolumeManager) CreateNbdServer(exporter, path string) error {
	cmdCreateNbsServer := fmt.Sprintf(`{ 'execute': 'nbd-server-start','arguments': { 'addr': { 'type': 'unix','data': { 'path': '%s/nbd.sock' }}}}`, path)
	cmdExportNbd := fmt.Sprintf(`{"execute":"nbd-server-add", "arguments":{"device":"imgfile", "name":"%s", "writable":true, "description":"%s exporter"}}`, exporter, exporter)

	cmds := []string{cmdCreateNbsServer,
		cmdExportNbd}
	for _, c := range cmds {
		if err := v.Monitor.ExecuteCommand(c); err != nil {
			return err
		}
	}
	return nil
}

func (v *VolumeManager) CreateVolume(image, id, size string) error {
	cmdBlockCreateFile := fmt.Sprintf(`{"execute": "blockdev-create", "arguments": {"job-id": "job0", "options": {"driver": "file", "filename": "%s", "size": 0}}}`, image)
	cmdJobDismiss := `{"execute": "job-dismiss", "arguments": {"id": "job0"}}`
	cmdBlockAddFile := fmt.Sprintf(`{"execute": "blockdev-add", "arguments": {"driver": "file", "filename": "%s", "node-name": "node-%s"}}`, image, id)
	cmdBlockCreateQCOW := fmt.Sprintf(`{"execute": "blockdev-create", "arguments": {"job-id": "job0", "options": {"driver": "qcow2", "file": "node-%s", "size": %s}}}`, id, size)
	cmds := []string{
		cmdBlockCreateFile,
		cmdJobDismiss,
		cmdBlockAddFile,
		cmdBlockCreateQCOW,
		cmdJobDismiss,
	}
	for _, c := range cmds {
		if err := v.Monitor.ExecuteCommand(c); err != nil {
			return err
		}
		if strings.Contains(c, "blockdev-create") {
			// HACK: implement loop to wait until job is completed and then dismiss it
			time.Sleep(3 * time.Second)

		}
	}

	return nil
}

func (v *VolumeManager) DeleteVolume(id string) error {
	c := fmt.Sprintf(` { "execute": "blockdev-del", "arguments": { "node-name": "node-%s" }`, id)
	if err := v.Monitor.ExecuteCommand(c); err != nil {
		return err
	}
	return nil
}

func (v *VolumeManager) ExposeVhostUser(id, vhostSock string) error {
	c := fmt.Sprintf(`{"execute": "block-export-add", "arguments": {"id": "vhost-%s", "node-name": "node-%s", "type": "vhost-user-blk", "writable": true, "addr": { "path": "%s", "type": "unix"}}}`, id, id, vhostSock)
	if err := v.Monitor.ExecuteCommand(c); err != nil {
		return err
	}
	return nil

}

func (v *VolumeManager) DeleteExporter(id string) error {
	c := fmt.Sprintf(`{"execute": "block-export-del", "arguments": {"id": "vhost-%s"}}`, id)
	if err := v.Monitor.ExecuteCommand(c); err != nil {
		return err
	}
	return nil

}
