package fakes

import "sync"

type ConfigWriter struct {
	WriteCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			LayerPath  string
			WorkingDir string
			CnbPath    string
		}
		Returns struct {
			String string
			Error  error
		}
		Stub func(string, string, string) (string, error)
	}
}

func (f *ConfigWriter) Write(param1 string, param2 string, param3 string) (string, error) {
	f.WriteCall.mutex.Lock()
	defer f.WriteCall.mutex.Unlock()
	f.WriteCall.CallCount++
	f.WriteCall.Receives.LayerPath = param1
	f.WriteCall.Receives.WorkingDir = param2
	f.WriteCall.Receives.CnbPath = param3
	if f.WriteCall.Stub != nil {
		return f.WriteCall.Stub(param1, param2, param3)
	}
	return f.WriteCall.Returns.String, f.WriteCall.Returns.Error
}
