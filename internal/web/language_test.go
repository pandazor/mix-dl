package web

import (
	"dl-manager/internal/model"
	"encoding/json"
	"reflect"
	"testing"
)

func TestLanguagePersistsWithoutChangingDownloads(t *testing.T) {
	a, _ := profileAPI(t)
	defer a.Close()
	before, _ := a.Store.Snapshot()
	for _, lang := range []string{"fr", "en", "es", "zh-CN", "ja"} {
		w := request(t, a, "POST", "settings/language", `{"language":"`+lang+`"}`)
		if w.Code != 200 {
			t.Fatalf("language %s: %d %s", lang, w.Code, w.Body.String())
		}
		w = request(t, a, "GET", "state", "")
		var body struct {
			State struct{ Settings map[string]any }
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.State.Settings["language"] != lang {
			t.Fatal("language not persisted", body.State.Settings)
		}
		now, _ := a.Store.Snapshot()
		b1, _ := json.Marshal(before.Settings)
		b2, _ := json.Marshal(now.Settings)
		var x, y map[string]any
		json.Unmarshal(b1, &x)
		json.Unmarshal(b2, &y)
		delete(x, "language")
		delete(y, "language")
		if !reflect.DeepEqual(x, y) {
			t.Fatal("language changed download settings")
		}
	}
	w := request(t, a, "POST", "settings/language", `{"language":"xx"}`)
	if w.Code == 200 {
		t.Fatal("unsupported language accepted")
	}
	w = request(t, a, "GET", "state", "")
	if !json.Valid(w.Body.Bytes()) {
		t.Fatal("bad state")
	}
	var old model.Settings
	b, _ := json.Marshal(before.Settings)
	if err := json.Unmarshal(b, &old); err != nil {
		t.Fatal(err)
	}
	if err := old.Validate(); err != nil {
		t.Fatal("legacy settings rejected", err)
	}
}
