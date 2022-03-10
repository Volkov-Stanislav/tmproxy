// Test Proxy package

// Tmproxy реализует прокси для вставки в тест сайта знака ТМ для слов длинной 6 символов.

// при запуске принимает следующие переменные командной строки:

// BaseURL string - если если переданное значение параметра не "" то оно дуте заменять HOST:PORT в запросе к серверу.

// LogLevel - уровень логирования, принимаются значения panic, error, warning, info, debug.

package main

import (
	"reflect"
	"testing"
)

func TestParseRespBody(t *testing.T) {
	type args struct {
		InText []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name:    "OK Test",
			args:    args{InText: []byte("<HEAD></HEAD><BODY><H1>Пример разбора ТеКсТа.<H1></BODY>")},
			want:    []byte("<HEAD></HEAD><BODY><H1>Пример\u2122 разбора ТеКсТа\u2122.<H1></BODY>"),
			wantErr: false,
		},
		{
			name:    "OK Test with Script",
			args:    args{InText: []byte("<HEAD><script>Пример разбора ТеКсТа.</script></HEAD><BODY><H1>Пример разбора ТеКсТа.<H1></BODY>")},
			want:    []byte("<HEAD><script>Пример разбора ТеКсТа.</script></HEAD><BODY><H1>Пример\u2122 разбора ТеКсТа\u2122.<H1></BODY>"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRespBody(tt.args.InText)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRespBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseRespBody() = %v, want %v", got, tt.want)
			}
		})
	}
}
