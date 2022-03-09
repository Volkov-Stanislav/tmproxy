// Test package

// Tmproxy реализует прокси для вставки в тест сайта знака ТМ для слов длинной 6 символов.
// при запуске принимает следующие переменные командной строки:
// BaseURL string - если переданный в запросе URL начинается с 'localhost' '127.0.0.1' то значение параматра будет подставляться вместо него.
// Timeout Int - значение максимального ожидания ответа сервера.
// LogLevel - уровень логирования, принимаются значения panic, error, warning, info, debug.
package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	baseURL  string
	timeout  int
	logLevel string
	Cl       http.Client
)

func main() {

	// логируем в виде текста
	log.SetFormatter(&log.TextFormatter{ForceColors: true, PadLevelText: true, FullTimestamp: false})
	// выгружаем в stdout вместо stderr
	log.SetOutput(os.Stdout)
	// Логируем по умолчанию от  уровня Debug, до разбора параметров командной строки
	log.SetLevel(log.DebugLevel)

	log.Debug("begin: main()")
	// разбираем параметры командной строки
	flag.StringVar(&baseURL, "BaseURL", "habrahabr.ru", "Base URL for replace, if \"\" no replace performed")
	flag.IntVar(&timeout, "Timeout", 10, "Timeout for get request")
	flag.StringVar(&logLevel, "log_level", "debug", "logging level, can be set to: panic, error, warning, info, debug.")
	flag.Parse()
	log.Debug("ok: flag.Parse()")

	switch logLevel {
	case "panic":
		log.SetLevel(log.PanicLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warning":
		log.SetLevel(log.WarnLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
		log.Info("Уровень логирования установлен в InfoLevel")
	case "debug":
		log.SetLevel(log.DebugLevel)
		log.Info("Уровень логирования установлен в DebugLevel")
	case "trace":
		log.SetLevel(log.TraceLevel)
		log.Info("Уровень логирования установлен в TraceLevel")
	default:
		log.SetLevel(log.WarnLevel)
		log.Error("log_level - неправильное значение параметра командной строки. Устанавливаем уровень логирования в Warning")
	}

	mux := http.NewServeMux()
	// Инициализируем обработчик
	rotQRfunc := http.HandlerFunc(PRHandler)

	// Биндим обработчик к точкам вызова
	mux.Handle("/", rotQRfunc)

	// Запускаем основной обработчик в отдельной горутине, что бы продолжить и отлавливать SIGTERM
	go func() {
		log.Debug("ok: http.Handle(/, PRHandler)")
		if errQr := http.ListenAndServe("127.0.0.1:8080", mux); errQr != nil {
			log.Error("Err: http.ListenAndServe(\"127.0.0.1:8080\") error: %s" + errQr.Error())
		}
	}()

	// Ждем сигнала завершения работы SIGTERM .
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)
	<-sigC
	log.Error("SIGTERM receided, exiting")
}

func PRHandler(w http.ResponseWriter, r *http.Request) {
	var CallURL string // URL сервера с которого получаем сайт

	startTime := time.Now()

	if len(baseURL) > 0 {
		CallURL = "http://" + baseURL + r.URL.Path
	} else {
		CallURL = "http://" + r.Host + r.URL.Path
	}
	log.Debugf("ok: Call received, CalledURL %s \n", CallURL)
	// формируем заголовок запроса к серверу, вставляем в него HTTP HEADERs  из запроса от браузера.
	CallReq, _ := http.NewRequest("GET", CallURL, nil)
	//CallReq.Header = r.Header
	for k, v := range r.Header {
		for val := range v {
			if k != "Connection" && k != "Keep-Alive" { //убираем заголовки подддержки удержания соединения и поддержки кодирования контента.
				CallReq.Header.Add(k, v[val])
				//		log.Debugf("ok: Request             Added Header | %s | %s \n", k, v[val])
			}
		}
	}
	CallReq.Header.Set("Connection", "close")
	CallReq.Header.Set("Keep-Alive", "timeout=0")

	result, err := Cl.Do(CallReq)
	if err != nil {
		log.Error("Call Failed" + err.Error())
		log.Error("err: Call to %s Failed with error: %s", CallURL, err.Error())
	}
	defer result.Body.Close()
	log.Debugf("ok: Call To Server Success, Duration = %s, CallUrl: | %s | \n", time.Since(startTime).String(), CallURL)

	// читаем ответ сервера, копируем в ответ полученные заголовки.
	for k, v := range result.Header {
		for val := range v {
			if k != "Connection" && k != "Keep-Alive" {
				w.Header().Add(k, v[val])
				log.Debugf("ok: Responce             Added Header | %s | %s \n", k, v[val])
			}
		}
	}
	w.Header().Set("Connection", "close")
	w.Header().Set("Keep-Alive", "timeout=0")

	var Body []byte
	// если в ответе HTML, то его надо обработать
	if strings.Contains(w.Header().Get("Content-Type"), "text/html") {
		if w.Header().Get("Content-Encoding") == "gzip" { // содержимое запаковано
			// предварительно распаковываем
			Body, _ = io.ReadAll(result.Body)
			gr, err := gzip.NewReader(bytes.NewBuffer(Body))
			if err != nil {
				log.Error("err: Error with gunzip create new reader: " + err.Error())
			}
			defer gr.Close()
			dataUnZip, err := ioutil.ReadAll(gr)
			if err != nil {
				log.Error("err: Error with read UnZIP: " + err.Error())
			}
			log.Debugf("ok: Unzippped: \n %s \n", string(dataUnZip))
			// Обрабатываем .
			ResParse, _ := ParseRespBody(dataUnZip)
			// тут пакуем обратно и пишем в ответ браузеру
			gw, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
			defer gw.Close()
			gw.Write(ResParse)
		} else { // содержимое не запаковано, обрабатываем сразу.
			Body, _ = io.ReadAll(result.Body)
			log.Debug("ok: Контент не упакован, сразу его полностью читаем.")
			ResParse, _ := ParseRespBody(Body)
			w.Write(ResParse)
		}
	} else {
		// не HTML передаем сразу без обработки
		Body, _ = io.ReadAll(result.Body)
		log.Debugf("ok: Received  Replay: \n %s\n ", string(Body))
		w.Write(Body)
	}
}

// ParseRespBody - обработчик ответа сервера, сканирует HTML, ищет выводимый текст и слова в нем длинна которых = 6 символам,
// и вставляет после найденных слов символ "\u2122
func ParseRespBody(InText []byte) (result []byte, err error) {

	return InText, nil
}
