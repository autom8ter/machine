package machine

import "errors"

// if a goroutine returns this error, every goroutines context will be cancelled
var Cancel = errors.New("[machine] cancel")
