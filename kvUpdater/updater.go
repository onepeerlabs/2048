package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"syscall/js"
	"time"

	"github.com/fairdatasociety/fairOS-dfs/pkg/contracts"
	"github.com/fairdatasociety/fairOS-dfs/pkg/dfs"
	"github.com/fairdatasociety/fairOS-dfs/pkg/logging"
	"github.com/fairdatasociety/fairOS-dfs/pkg/user"
	"github.com/sirupsen/logrus"
)

var (
	ctx    context.Context
	cancel context.CancelFunc

	api *dfs.API
	ui  *user.Info

	username    string
	password    string
	podName     string
	tableName   string
	beeEndpoint string
	stampId     string
	rpc         string
	scoreKey    = "bestScore"
	visitKey    = "visitCount"

	lastUpdated = time.Time{}
)

func main() {
	js.Global().Set("addVisitor", js.FuncOf(addVisitor))
	js.Global().Set("updateHighScore", js.FuncOf(updateHighScore))
	js.Global().Set("stop", js.FuncOf(stop))
	ctx, cancel = context.WithCancel(context.Background())
	<-ctx.Done()
}

func addVisitor(_ js.Value, funcArgs []js.Value) interface{} {
	baseURL := js.Global().Get("location").Get("href").String()

	handler := js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		resolve := args[0]
		reject := args[1]
		if baseURL == "http://localhost:3000" {
			reject.Invoke("operation not permitted")
			return nil
		}
		if len(funcArgs) != 1 {
			reject.Invoke("not enough arguments. \"addVisitor(visitor)\"")
			return nil
		}

		config, _ := contracts.TestnetConfig()
		config.ProviderBackend = rpc
		logger := logging.New(os.Stdout, logrus.ErrorLevel)

		go func() {
			var err error
			api, err = dfs.NewDfsAPI(
				beeEndpoint,
				stampId,
				config,
				nil,
				logger,
			)
			if err != nil {
				reject.Invoke(fmt.Sprintf("failed to connect to fairOS: %s", err.Error()))
				return
			}
			ui, _, _, err = api.LoginUserV2(username, password, "")
			if err != nil {
				reject.Invoke(fmt.Sprintf("Failed to login user : %s", err.Error()))
				api = nil
				return
			}

			_, err = api.OpenPod(podName, ui.GetSessionId())
			if err != nil {
				reject.Invoke(fmt.Sprintf("podOpen failed : %s", err.Error()))
				api = nil
				return
			}

			err = api.KVOpen(ui.GetSessionId(), podName, tableName)
			if err != nil {
				reject.Invoke(fmt.Sprintf("kvOpen failed : %s", err.Error()))
				api = nil
				return
			}

			visitCount := funcArgs[0].Int()
			value := []byte(strconv.Itoa(visitCount))
			err = api.KVPut(ui.GetSessionId(), podName, tableName, visitKey, value)
			if err != nil {
				reject.Invoke(fmt.Sprintf("kvEntryPut failed : %s", err.Error()))
				return
			}
			fmt.Println("visitor updated")
			resolve.Invoke("success")
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

func updateHighScore(_ js.Value, funcArgs []js.Value) interface{} {
	baseURL := js.Global().Get("location").Get("href").String()

	handler := js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		resolve := args[0]
		reject := args[1]
		if baseURL == "http://localhost:3000" {
			reject.Invoke("operation not permitted")
			return nil
		}
		if len(funcArgs) != 1 {
			reject.Invoke("not enough arguments. \"updateHighScore(score)\"")
			return nil
		}

		if time.Since(lastUpdated) < time.Minute {
			reject.Invoke("too frequent update")
			return nil
		}
		lastUpdated = time.Now()
		go func() {
			score := funcArgs[0].Int()
			if api == nil {
				config, _ := contracts.TestnetConfig()
				config.ProviderBackend = rpc
				logger := logging.New(os.Stdout, logrus.DebugLevel)

				var err error
				api, err = dfs.NewDfsAPI(
					beeEndpoint,
					stampId,
					config,
					nil,
					logger,
				)
				if err != nil {
					reject.Invoke(fmt.Sprintf("failed to connect to fairOS: %s", err.Error()))
					return
				}
				ui, _, _, err = api.LoginUserV2(username, password, "")
				if err != nil {
					reject.Invoke(fmt.Sprintf("Failed to login user : %s", err.Error()))
					return
				}

				_, err = api.OpenPod(podName, ui.GetSessionId())
				if err != nil {
					reject.Invoke(fmt.Sprintf("podOpen failed : %s", err.Error()))
					return
				}

				err = api.KVOpen(ui.GetSessionId(), podName, tableName)
				if err != nil {
					reject.Invoke(fmt.Sprintf("kvOpen failed : %s", err.Error()))
					return
				}
			}

			value := []byte(strconv.Itoa(score))
			err := api.KVPut(ui.GetSessionId(), podName, tableName, scoreKey, value)
			if err != nil {
				reject.Invoke(fmt.Sprintf("kvEntryPut failed : %s", err.Error()))
				return
			}
			resolve.Invoke("success")
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

func stop(js.Value, []js.Value) interface{} {
	cancel()
	return nil
}
