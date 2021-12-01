package controller

import (
	"encoding/json"
	"testing"

	log "github.com/sirupsen/logrus"
)

func Test_getLocalStrategies(t *testing.T) {
	s, err := getLocalStrategies("../../conf/strategies.yaml")
	if err != nil {
		t.Errorf("Test getLocalStrategies err: %+v", err)
		return
	}
	bytes, err := json.Marshal(s)
	if err != nil {
		t.Errorf("Json marshal err: %v", err)
		return
	}
	log.Printf("Get local strategies: %s", bytes)
}

func Test_parseStartTime(t *testing.T) {
	type args struct {
		validTime string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"test1", args{validTime: "0:00-09:30"}, "0 00 0 * * ?", false},
		{"test2", args{validTime: "9:30-20:00"}, "0 30 9 * * ?", false},
		{"test3", args{validTime: "20:00-24:00"}, "0 00 20 * * ?", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := genStartTimeSpec(tt.args.validTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("genStartTimeSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("genStartTimeSpec() got = %v, want %v", got, tt.want)
			}
		})
	}
}
