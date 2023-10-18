package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/moqsien/gogpt/pkgs/config"
	"github.com/moqsien/gogpt/pkgs/gpt"
)

type ItemList struct {
	List []map[string]string `json:"list"`
}

func GetPrompts() {
	itemList := ItemList{List: []map[string]string{}}
	d := `C:\Users\moqsien\data\projects\md\ChatGPT-System-Prompts\prompts`
	dList, _ := os.ReadDir(d)
	for _, item := range dList {
		if item.IsDir() {
			dd := filepath.Join(d, item.Name())
			l, _ := os.ReadDir(dd)
			for _, entry := range l {
				if !entry.IsDir() {
					fPath := filepath.Join(dd, entry.Name())
					content, _ := os.ReadFile(fPath)
					if len(content) > 0 {
						sList := strings.Split(string(content), "\n")
						act := strings.ReplaceAll(entry.Name(), ".md", "")
						act = strings.ReplaceAll(act, "-", "_")
						for _, s := range sList {
							if len(s) > 40 {
								itemList.List = append(itemList.List, map[string]string{
									"act":    act,
									"prompt": strings.TrimSuffix(s, "\r"),
								})
							}
						}
					}
				}
			}
		}
	}
	result, _ := json.MarshalIndent(itemList, "", "    ")
	os.WriteFile("result.json", result, os.ModePerm)
}

func main() {
	// GetPrompts()
	cnf := config.GetDefaultConfig()
	p := gpt.NewGPTPrompt(cnf)
	p.ChoosePrompt()
	fmt.Println(p.PromptStr())
}