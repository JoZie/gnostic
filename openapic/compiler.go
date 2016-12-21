// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate ./COMPILE-PROTOS.sh

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/golang/protobuf/proto"
	"github.com/googleapis/openapi-compiler/OpenAPIv2"
	"github.com/googleapis/openapi-compiler/compiler"
	plugins "github.com/googleapis/openapi-compiler/plugins"
)

func main() {
	var textProtoFileName = flag.String("text_out", "", "Write a text proto to a file with the specified name.")
	var jsonProtoFileName = flag.String("json_out", "", "Write a json proto to a file with the specified name.")
	var binaryProtoFileName = flag.String("pb_out", "", "Write a binary proto to a file with the specified name.")
	var errorFileName = flag.String("errors_out", "", "Write compilation errors to a file with the specified name.")
	var keepReferences = flag.Bool("keep_refs", false, "Disable resolution of $ref references.")

	var pluginName = flag.String("plugin", "", "Run the specified plugin (for development only).")

	flag.Parse()

	flag.Usage = func() {
		fmt.Printf("Usage: openapic [OPTION] OPENAPI_FILE\n")
		fmt.Printf("OPENAPI_FILE is the path to the input OpenAPI " +
			"file to parse.\n")
		fmt.Printf("Output is generated based on the options given:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	var input string

	if len(flag.Args()) == 1 {
		input = flag.Arg(0)
	} else {
		flag.Usage()
		return
	}

	if *textProtoFileName == "" &&
		*jsonProtoFileName == "" &&
		*binaryProtoFileName == "" &&
		*errorFileName == "" &&
		*pluginName == "" {
		fmt.Printf("Missing output directives.\n")
		flag.Usage()
		return
	}

	raw, err := compiler.ReadFile(input)
	if err != nil {
		fmt.Printf("Error: No Specification.\n%+v\n", err)
		os.Exit(-1)
	}

	document, err := openapi_v2.NewDocument(raw, compiler.NewContext("$root", nil))
	if err != nil {
		fmt.Printf("%+v\n", err)
		if *errorFileName != "" {
			ioutil.WriteFile(*errorFileName, []byte(err.Error()), 0644)
		}
		os.Exit(-1)
	}

	if !*keepReferences {
		_, err = document.ResolveReferences(input)
		if err != nil {
			fmt.Printf("%+v\n", err)
			if *errorFileName != "" {
				ioutil.WriteFile(*errorFileName, []byte(err.Error()), 0644)
			}
			os.Exit(-1)
		}
	}

	if *textProtoFileName != "" {
		ioutil.WriteFile(*textProtoFileName, []byte(proto.MarshalTextString(document)), 0644)
		fmt.Printf("Output protobuf textfile: %s\n", *textProtoFileName)
	}
	if *jsonProtoFileName != "" {
		jsonBytes, _ := json.Marshal(document)
		ioutil.WriteFile(*jsonProtoFileName, jsonBytes, 0644)
		fmt.Printf("Output protobuf json file: %s\n", *jsonProtoFileName)
	}
	if *binaryProtoFileName != "" {
		protoBytes, _ := proto.Marshal(document)
		ioutil.WriteFile(*binaryProtoFileName, protoBytes, 0644)
		fmt.Printf("Output protobuf binary file: %s\n", *binaryProtoFileName)
	}

	if *pluginName != "" {
		request := &plugins.PluginRequest{}
		request.Parameter = ""

		version := &plugins.Version{}
		version.Major = 0
		version.Minor = 1
		version.Patch = 0
		request.CompilerVersion = version

		wrapper := &plugins.Wrapper{}
		wrapper.Name = input
		wrapper.Version = "v2"
		protoBytes, _ := proto.Marshal(document)
		wrapper.Value = protoBytes
		request.Wrapper = []*plugins.Wrapper{wrapper}
		requestBytes, _ := proto.Marshal(request)

		cmd := exec.Command("openapi_" + *pluginName)
		cmd.Stdin = bytes.NewReader(requestBytes)
		output, err := cmd.Output()
		if err != nil {
			fmt.Printf("Error: %+v\n", err)
		}
		response := &plugins.PluginResponse{}
		err = proto.Unmarshal(output, response)
		for _, text := range response.Text {
			os.Stdout.Write([]byte(text))
		}
	}
}
