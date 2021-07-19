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

var (
	QsdBin           = "/usr/local/bin/qemu-storage-daemon"
	QsdSDirContainer = "/qsd"
)

func QsdArgs(pathSocket string) []string {
	return []string{
		"--chardev",
		fmt.Sprintf("socket,server=on,path=%s/qmp.sock,id=chardev0,nowait", pathSocket),
		"--monitor",
		"chardev=chardev0",
	}
}

type VolumeManager struct {
	Monitor *QMPMonitor
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
	cmdBlockAddFile := fmt.Sprintf(`{"execute": "blockdev-add", "arguments": {"driver": "file", "filename": "%s", "node-name": "%s"}}`, image, id)
	cmdBlockCreateQCOW := fmt.Sprintf(`{"execute": "blockdev-create", "arguments": {"job-id": "job0", "options": {"driver": "qcow2", "file": "%s", "size": %s}}}`, size, id)
	cmds := []string{
		cmdBlockCreateFile,
		cmdJobDismiss,
		cmdBlockAddFile,
		cmdBlockCreateQCOW,
		cmdJobDismiss,
	}
	for _, c := range cmds {
		if strings.Contains(c, "job-dismiss") {
			// HACK: implement loop to wait until job is completed and then dismiss it
			time.Sleep(2 * time.Second)

		}
		if err := v.Monitor.ExecuteCommand(c); err != nil {
			return err
		}
	}

	return nil
}

func (v *VolumeManager) ExposeVhostUser(id, path string) error {
	vhostSock := fmt.Sprintf("%s/vhost-user.sock", path)
	cmdExport := fmt.Sprintf(`{"block-export-add": "addr": "%s", "id": "%s", "node-name": "%s", "type": "vhost-user-blk"}`, vhostSock, id, id)
	cmds := []string{
		cmdExport,
	}
	for _, c := range cmds {
		if err := v.Monitor.ExecuteCommand(c); err != nil {
			return err
		}
	}
	return nil

}
