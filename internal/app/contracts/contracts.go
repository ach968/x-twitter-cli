package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	app "github.com/ach968/x-twitter-cli/internal/app"
)

var requiredOperations = []app.OperationName{
	app.SearchTimeline,
	app.Bookmarks,
	app.BookmarkSearchTimeline,
}

var supportedOperations = map[app.OperationName]struct{}{
	app.HomeTimeline:           {},
	app.SearchTimeline:         {},
	app.Bookmarks:              {},
	app.BookmarkSearchTimeline: {},
	app.TweetDetail:            {},
}

const invalidOperationsMessage = "Contract properties must contain every required operation and no unsupported operations"

type ContractPropertiesError struct {
	Code    string
	Message string
}

func (e *ContractPropertiesError) Error() string { return e.Message }

func newContractPropertiesError(message string, code ...string) error {
	value := "INVALID_CONTRACT_PROPERTIES"
	if len(code) > 0 {
		value = code[0]
	}
	return &ContractPropertiesError{Code: value, Message: message}
}

func isXOwnedHost(host string) bool {
	return host == "x.com" || strings.HasSuffix(host, ".x.com")
}

func validateOperation(operation app.OperationContract) bool {
	validMethod := operation.Method == "GET" || operation.Method == "POST"
	validEncoding := (operation.Method == "GET" && operation.Encoding == "query") ||
		(operation.Method == "POST" && operation.Encoding == "json" && operation.Body != nil)
	return operation.Family == "graphql" &&
		isXOwnedHost(operation.Host) &&
		strings.HasPrefix(operation.Path, "/") &&
		validMethod && validEncoding &&
		operation.Variables != nil && operation.Features != nil && operation.FieldToggles != nil
}

func decodeOperationObject(contents []byte) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(contents))
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return nil, errors.New("operations must be an object")
	}
	operations := map[string]json.RawMessage{}
	for decoder.More() {
		name, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := name.(string)
		if !ok {
			return nil, errors.New("operation name must be a string")
		}
		if _, duplicate := operations[key]; duplicate {
			return nil, errors.New("duplicate operation name")
		}
		var contract json.RawMessage
		if err := decoder.Decode(&contract); err != nil {
			return nil, err
		}
		operations[key] = contract
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	return operations, nil
}

func Load(path string) (app.ContractProperties, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return app.ContractProperties{}, newContractPropertiesError(
			fmt.Sprintf("Unable to read contract properties from %s", path),
			"CONTRACT_PROPERTIES_UNREADABLE",
		)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(contents, &raw); err != nil || raw == nil {
		return app.ContractProperties{}, newContractPropertiesError("Contract properties are not valid JSON")
	}

	var version int
	if err := json.Unmarshal(raw["version"], &version); err != nil || version != 1 {
		return app.ContractProperties{}, newContractPropertiesError(
			"Unsupported contract properties version",
			"UNSUPPORTED_CONTRACT_VERSION",
		)
	}

	operationRaw, err := decodeOperationObject(raw["operations"])
	if err != nil || operationRaw == nil {
		return app.ContractProperties{}, newContractPropertiesError("Operations must be an object")
	}
	if len(operationRaw) < len(requiredOperations) || len(operationRaw) > len(supportedOperations) {
		return app.ContractProperties{}, newContractPropertiesError(invalidOperationsMessage)
	}

	properties := app.ContractProperties{Version: version, Operations: make(map[app.OperationName]app.OperationContract, len(operationRaw))}
	for _, name := range requiredOperations {
		if _, ok := operationRaw[string(name)]; !ok {
			return app.ContractProperties{}, newContractPropertiesError(invalidOperationsMessage)
		}
	}
	for rawName, encoded := range operationRaw {
		name := app.OperationName(rawName)
		if _, ok := supportedOperations[name]; !ok {
			return app.ContractProperties{}, newContractPropertiesError(invalidOperationsMessage)
		}
		var operation app.OperationContract
		if err := json.Unmarshal(encoded, &operation); err != nil || !validateOperation(operation) {
			return app.ContractProperties{}, newContractPropertiesError(invalidOperationsMessage)
		}
		properties.Operations[name] = operation
	}
	return properties, nil
}

func ErrorCode(err error) string {
	var target *ContractPropertiesError
	if errors.As(err, &target) {
		return target.Code
	}
	return ""
}

func UnavailableFailure() *app.OperationFailure {
	return &app.OperationFailure{
		Code:            "CONTRACT_FAILED",
		Message:         "Operation contracts are unavailable or invalid",
		RecoveryCommand: "twt contract refresh",
	}
}
