package qsd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

func (q *QMPMonitor) ExecuteCommandRaw(qmpCmd string) ([]byte, error) {
	cmd := []byte(qmpCmd)
	fmt.Printf("Executed command %s\n", qmpCmd)
	raw, err := q.monitor.Run(cmd)
	if err != nil {
		return raw, fmt.Errorf("failed running qmp command %s: %v", qmpCmd, err)
	}
	fmt.Printf("result: %s\n", string(raw))
	return raw, nil

}

func (q *QMPMonitor) ExecuteCommand(qmpCmd string) error {
	raw, err := q.ExecuteCommandRaw(qmpCmd)
	if err != nil {
		return err
	}
	var result statusResult
	err = json.Unmarshal(raw, &result)
	if err != nil {
		return fmt.Errorf("failed parsing result %v", err)
	}

	fmt.Println(result.Return.Status)
	return nil
}

func (q *QMPMonitor) Events() (<-chan qmp.Event, error) {
	return q.monitor.Events()
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

func (v *VolumeManager) dismissJob(id string) error {
	cmdJobDismiss := fmt.Sprintf(`{
  "execute": "job-dismiss",
  "arguments": {
    "id": "%s"
  }
}`, id)
	if err := v.Monitor.ExecuteCommand(cmdJobDismiss); err != nil {
		return err
	}
	chEvents, err := v.Monitor.Events()
	if err != nil {
		return fmt.Errorf("Failed creating monitor event %v", err)
	}
	for {
		select {
		case <-time.After(time.Second * 10):
			return fmt.Errorf("Timeout in dismissing job %s", id)
		case event := <-chEvents:
			fmt.Printf("Events %v \n", event)
			if event.Event == "BLOCK_JOB_COMPLETED" {
				fmt.Printf("Dismissed job %s \n", id)
				return nil
			}
		}
	}
	return nil
}

func (v *VolumeManager) createImage(image, id, size, format string) error {
	// if the image already exists do not recreate
	if _, err := os.Stat(image); os.IsNotExist(err) {
		cmd := exec.Command("qemu-img", "create", "-f", format, image, size)
		stdoutStderr, err := cmd.CombinedOutput()
		fmt.Printf("execute: qemu-img output: %s \n", stdoutStderr)
		if err != nil {
			return fmt.Errorf("qemu-img failed err:%v", stdoutStderr, err)
		}
	}
	cmdBlockAddFile := fmt.Sprintf(`{
  "execute": "blockdev-add",
  "arguments": {
    "driver": "file",
    "filename": "%s",
    "node-name": "node-%s"
  }
}`, image, id)

	if err := v.Monitor.ExecuteCommand(cmdBlockAddFile); err != nil {
		return err
	}
	return nil
}

func (v *VolumeManager) CreateVolume(image, id, size string) error {
	return v.createImage(image, id, size, "qcow2")
}

func (v *VolumeManager) DeleteVolume(id string) error {
	c := fmt.Sprintf(`{
  "execute": "blockdev-del",
  "arguments": {
    "node-name": "node-%s"}}`, id)
	if err := v.Monitor.ExecuteCommand(c); err != nil {
		return err
	}
	return nil
}

func (v *VolumeManager) ExposeVhostUser(id, vhostSock string) error {
	c := fmt.Sprintf(`{
  "execute": "block-export-add",
  "arguments": {
    "id": "vhost-%s",
    "node-name": "node-%s",
    "type": "vhost-user-blk",
    "writable": true,
    "addr": {
      "path": "%s",
      "type": "unix"
    }
  }
}`, id, id, vhostSock)
	if err := v.Monitor.ExecuteCommand(c); err != nil {
		return err
	}
	return nil

}

func (v *VolumeManager) DeleteExporter(id string) error {
	c := fmt.Sprintf(`{
  "execute": "block-export-del",
  "arguments": {
    "id": "vhost-%s"
  }
}`, id)
	if err := v.Monitor.ExecuteCommand(c); err != nil {
		return err
	}
	return nil

}

func (v *VolumeManager) CreateSnapshotWithBackingNode(imageID, snapshotID, image, snapshot, backing string) error {

	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", "-F", "qcow2", "-b", image, snapshot)
	stdoutStderr, err := cmd.CombinedOutput()
	fmt.Printf("execute: qemu-img output: %s \n", stdoutStderr)
	if err != nil {
		return fmt.Errorf("%v failed output: %s err:%v", cmd, stdoutStderr, err)
	}
	cmdBlockAdd := fmt.Sprintf(`{
  "execute": "blockdev-add","arguments": {
    "driver": "qcow2",
    "file": {"driver": "file","filename": "%s"},
    "backing": "node-%s",
    "node-name": "node-%s"}}`, snapshot, backing, snapshotID)
	return v.Monitor.ExecuteCommand(cmdBlockAdd)
}

func (v *VolumeManager) CreateSnapshot(imageID, snapshotID, image, snapshot string) error {
	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", "-F", "qcow2", "-b", image, snapshot)
	stdoutStderr, err := cmd.CombinedOutput()
	fmt.Printf("execute: qemu-img output: %s \n", stdoutStderr)
	if err != nil {
		return fmt.Errorf("%v failed output: %s err:%v", cmd, stdoutStderr, err)
	}
	cmdBlockAdd := fmt.Sprintf(`{
  "execute": "blockdev-add","arguments": {
    "driver": "qcow2",
    "file": {"driver": "file","filename": "%s"},
    "backing": null,
    "node-name": "node-%s"}}`, snapshot, snapshotID)

	cmdBlockSnap := fmt.Sprintf(`{
  "execute": "blockdev-snapshot",
  "arguments": {
    "node": "node-%s",
    "overlay": "node-%s"}}`, imageID, snapshotID)
	cmds := []string{
		cmdBlockAdd,
		cmdBlockSnap,
	}

	for _, c := range cmds {
		if err := v.Monitor.ExecuteCommand(c); err != nil {
			return err
		}
	}
	return nil
}

func (v *VolumeManager) StreamImage(base, overlay string) error {
	jobID := "job0"
	cmdBlockstream := fmt.Sprintf(`{
    "execute": "block-stream",
    "arguments": {
        "device": "node-%s",
        "job-id": "%s",
	"base-node": "node-%s"}}`, overlay, jobID, base)
	if err := v.Monitor.ExecuteCommand(cmdBlockstream); err != nil {
		return err
	}
	return v.dismissJob(jobID)
}

func (v *VolumeManager) CommitImage(node, top, base string) error {
	jobID := "job0"
	cmdBlockstream := fmt.Sprintf(`{
    "execute": "block-commit",
    "arguments": {
        "device": "node-%s",
        "job-id": "%s",
	"top": "%s",
        "base": "%s"}}`, node, jobID, top, base)
	return v.Monitor.ExecuteCommand(cmdBlockstream)
}

type ImageInfo struct {
	Filename              string           `json:"filename"`
	Format                string           `json:"format"`
	VirtualSize           int              `json:"virtual-size"`
	BackingFile           string           `json:"backing_file"`
	FullBackingFilename   string           `json:"full-backing-filename"`
	BackingFilenameFormat string           `json:"backing-filename-format"`
	Snapshots             []SnapshotInfo   `json:"snapshots"`
	BackingImage          BackingImageInfo `json:"backing-image"`
}

type BackingImageInfo struct {
	Filename    string `json:"filename"`
	Format      string `json:"format"`
	VirtualSize int    `json:"virtual-size"`
}

type SnapshotInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	VMStateSize int    `json:"vm-state-size"`
	DateSec     int    `json:"date-sec"`
	DateNsec    int    `json:"date-nsec"`
	VMClockSec  int    `json:"vm-clock-sec"`
	VMClockNsec int    `json:"vm-clock-nsec"`
}

type NameBlockNode struct {
	Ro               bool      `json:"ro"`
	Drv              string    `json:"drv"`
	Encrypted        bool      `json:"encrypted"`
	File             string    `json:"file"`
	NodeName         string    `json:"node-name"`
	BackingFileDepth int       `json:"backing_file_depth"`
	Bps              int       `json:"bps"`
	BpsRd            int       `json:"bps_rd"`
	BpsWr            int       `json:"bps_wr"`
	Iops             int       `json:"iops"`
	IopsRd           int       `json:"iops_rd"`
	IopsWr           int       `json:"iops_wr"`
	BpsMax           int       `json:"bps_max"`
	BpsRdMax         int       `json:"bps_rd_max"`
	BpsWrMax         int       `json:"bps_wr_max"`
	IopsMax          int       `json:"iops_max"`
	IopsRdMax        int       `json:"iops_rd_max"`
	IopsWrMax        int       `json:"iops_wr_max"`
	IopsSize         int       `json:"iops_size"`
	WriteThreshold   int       `json:"write_threshold"`
	Image            ImageInfo `json:"image"`
}

type QueryNameBlockNodesReturn struct {
	ID     string          `json:"id"`
	Return []NameBlockNode `json:"return"`
}

func (v *VolumeManager) GetNameBlockNodes() ([]NameBlockNode, error) {
	cmdQueryNamedBlockNodes := `{ "execute": "query-named-block-nodes" }`
	raw, err := v.Monitor.ExecuteCommandRaw(cmdQueryNamedBlockNodes)
	if err != nil {
		return []NameBlockNode{}, err
	}
	var result QueryNameBlockNodesReturn
	err = json.Unmarshal(raw, &result)
	if err != nil {
		return []NameBlockNode{}, fmt.Errorf("failed parsing result %v", err)
	}

	return result.Return, nil
}
