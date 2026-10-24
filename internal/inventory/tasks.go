package inventory

import (
	"encoding/json"
	"strings"
)

func folderOpenCommands(data []byte) ([]string, error) {
	var doc struct {
		Tasks []struct {
			Command    string `json:"command"`
			RunOptions struct {
				RunOn string `json:"runOn"`
			} `json:"runOptions"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	var out []string
	for _, task := range doc.Tasks {
		if !strings.EqualFold(task.RunOptions.RunOn, "folderOpen") {
			continue
		}
		cmd := strings.TrimSpace(task.Command)
		if cmd == "" {
			continue
		}
		out = append(out, cmd)
	}
	return out, nil
}
