/*
	this package makes requests to the ChatGPT API and writes the responses to .txt log files
	global constants are used to:
		- configure the requests
			- adjust endpoint to call
			- adjust body of request
		- configure how many requests are made
			- how many files are created
			- how many request/responses are written to each file

	add your API key (will have compile error until then), adjust request inputs, and see logs generated in the `logs` directory
*/
package main
  
import (
	"fmt"
	"io/ioutil"
	"net/http"
	"log"
	"strings"
	"encoding/json"
	"path/filepath"
	"os"
)
// structs to unwrap the json of responses into objects
type GeneralResponse struct {
	Model string
	ResponseContent string
}

type ChatCompletion struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Usage struct {
		PromptTokens    int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens     int `json:"total_tokens"`
	} `json:"usage"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
	} `json:"choices"`
}

type Edits struct {
	Object string `json:"object"`
	Created int `json:"created"`
	Choices []struct {
		Text string `json:"text"`
		Index int `json:"index"`
	} `json:"choices"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens     int `json:"total_tokens"`
	} `json:"usage"`
}

// add your API key
const API_KEY = 

//	setting to `true` will cause the program to log the request to the terminal, and then terminate before sending the request to ChatGPT
const dryFire = false

//	request configurations

//	specify which endpoint to call by setting `endpointSpecifier` to the corresponding integer key
var endpoints = map[int]string {
	1:"https://api.openai.com/v1/chat/completions",
	2:"https://api.openai.com/v1/edits",
}
var endpointSpecifier int = 1

/*	regardless of what endpoint you are using, this will be the prompt to send
	if you are using the /edits endpoint, this will be the `instruction` that is passed in*/
const prompt = "write me a hello world program in go"
const model = "gpt-3.5-turbo"
const messagesRole = "user"

//	if making a request to the /edits endpoint, put the desired `input` into this map at the "input" key
var editsReq = map[string]string {
	"model":"code-davinci-edit-001",
	"input":"service: my-service\n provider:\n     name: aws\n     runtime: nodejs12.x\nregion: us-east-1\n     resources:\n      Resources:\n    PromptsTable:\n           Type: AWS::DynamoDB::Table\n          Properties:\n                 TableName: prompts\n              AttributeDefinitions:\n- AttributeName: id\n                     AttributeType: S\n              StreamSpecification:\nStreamViewType: NEW_AND_OLD_IMAGES\n              KeySchema:\n                  - AttributeName: id\n             KeyType: HASH\n                 BillingMode: PAY_PER_REQUEST",
	"instruction":prompt,
}

/*	request "knob" options are 'temperature' and 'top_p'
		specify which knob to tweak, it's initial (min) value, it's max value, and by what value to increment each subsequent request (Increment)
		Together, these data determine how many requests will be made and written to each log file.
			e.g. min = 0.0, max = 0.8, Increment = 0.2,  this will be 5 requests – the "knob" metric will be incremented by +0.2, until it goes from 0.0 to 0.8
*/
const knob = "temperature"
const knobTuneMin = 0.6
const knobTuneMax = 1
const knobTuneIncrement = 0.2

/*	how many log files to create, where each file has as many requests written to it as the "knob" variables determine
		e.g. given the example above and a `logSampleSize` of 2, this will create 2 files and both will have 5 requests written to them
	note: the requests are NOT written to both files at once, but rather 5 requests will be made and written to log 01, and then an
		additional 5 requests will be made and written to log 02
*/
const logSampleSize = 2


/* this is a summary of the request you are making to ChatGPT and it will also serve as the parent directory for the log files
		NOTE: logs will be placed in a sub-directory of this file that will be named with whatever value found in the `knob` constant above
			e.g.  hello-world-program/top_p/01-api-log_top_p.text
*/
const promptFileTitle = "hello-world-program"
const fileSeperator = "_______________________________________" // for formatting log files; don't bother about it

// entry point of the program
func main() {
	executeRequests()
}


func executeRequests() {
	method := "POST"

	for j := 1; j <= logSampleSize; j++ {
		var filePrefix = fmt.Sprintf("%0.2d", j)
		for knobTune := knobTuneMin; knobTune <= knobTuneMax; knobTune = knobTune + knobTuneIncrement {

			var payload *strings.Reader
			switch endpointSpecifier {
				case 1:
					payload = chatCompletionRequest(float32(knobTune))
				case 2:
					payload = editsRequest(float32(knobTune))
			}


			client := &http.Client {}
			req, err := http.NewRequest(method, endpoints[endpointSpecifier], payload)

			if err != nil {
				fmt.Println(err)
				return
			}
			req.Header.Add("Authorization", "Bearer " + API_KEY)
			req.Header.Add("Content-Type", "application/json")

			//	log request to console and stop execution
			if dryFire {
				//	make sure data is large enough to hold your whole request; 4000 bytes here
				data := make([]byte, 4000)
				req.Body.Read(data)
				fmt.Println(string(data))
				panic("end")
			}

			res, err := client.Do(req)
			if err != nil {
				fmt.Println(err)
				return
			}
			defer res.Body.Close()

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				fmt.Println(err)
				return
			}

			response := handleResponseBody(string(body))
			
			writeResponse(fmt.Sprintf("%s\nprompt: %s\n", fileSeperator, prompt), filePrefix)
			writeResponse(fmt.Sprintf("%s: %.1f\n",knob, knobTune), filePrefix)
			writeResponse(fmt.Sprintf("endpoint: %s\n", endpoints[endpointSpecifier]), filePrefix)
			writeResponse(fmt.Sprintf("model: %s\n", response.Model), filePrefix)
			writeResponse(fmt.Sprintf("messages.role: %s\n\n\nobservations:\n%s\n", messagesRole, fileSeperator), filePrefix)
			writeResponse(fmt.Sprintf("\nresponse:\n\n%s\n\n\n\n", response.ResponseContent), filePrefix)
		}
	}
}

func chatCompletionRequest(knobTune float32) *strings.Reader {
	payload := strings.NewReader(fmt.Sprintf(`{"model": "%s","messages":[{"role":"%s", "content":"%s"}],"%s":%.1f}`,
			model, messagesRole, strings.ReplaceAll(prompt,"\n","\\n"), knob, knobTune))
	return payload
}

// creates the payload for a request to the /edits endpoint
func editsRequest(knobTune float32) *strings.Reader {
	payload := strings.NewReader(fmt.Sprintf(`{"model":"%s","input":"%s","instruction":"%s","%s":%.1f}`,
			editsReq["model"], strings.ReplaceAll(editsReq["input"], "\n", "\\n"), editsReq["instruction"], knob, knobTune))
	return payload
}

// takes response string and decides which method to use for unwrapping the json into a usable object
func handleResponseBody(respString string) GeneralResponse {
	switch endpointSpecifier {
	case 1:
		return unmarshalChatCompletion(respString)
	case 2:
		return unmarshalEdit(respString)
	}
	return GeneralResponse{"nil", "nil"}
}

// unwraps responsense coming from the /chat/completions endpoint
func unmarshalEdit(data string) GeneralResponse {
	respStruct := Edits{}

	fmt.Println("data: " + data)

	err := json.Unmarshal([]byte(data), &respStruct)
	if err != nil {
		log.Fatal(err)
	}

	return GeneralResponse{
		editsReq["model"],
		respStruct.Choices[0].Text,
	}
}

// unwraps responsense coming from the /chat/completions endpoint
func unmarshalChatCompletion(data string) GeneralResponse {
	respStruct := ChatCompletion{}

	err := json.Unmarshal([]byte(data), &respStruct)
	if err != nil {
		log.Fatal(err)
	}

	return GeneralResponse{
		model,
		respStruct.Choices[0].Message.Content,
	}
}

func writeResponse(content string, prfx string) {
	path := "logs/" + promptFileTitle + "/" + knob + "/" + prfx + "-api-log_" + knob + ".txt"

	// create directories if not exists
	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		fmt.Println("Error:", err)
	}

	// open log file
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}


	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
}
