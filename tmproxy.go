// Test Proxy package

// Tmproxy реализует прокси для вставки в тест сайта знака ТМ для слов длинной 6 символов.
// при запуске принимает следующие переменные командной строки:
// BaseURL string - если если переданное значение параметра не "" то оно дуте заменять HOST:PORT в запросе к серверу.
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
	logLevel string
	Cl       http.Client
)

func main() {

	// логируем в виде текста
	log.SetFormatter(&log.TextFormatter{ForceColors: true, PadLevelText: true, FullTimestamp: false})
	// выгружаем в stdout вместо stderr
	log.SetOutput(os.Stdout)
	// Логируем по умолчанию от  уровня Error, до разбора параметров командной строки
	log.SetLevel(log.ErrorLevel)

	log.Debug("begin: main()")
	// разбираем параметры командной строки
	flag.StringVar(&baseURL, "BaseURL", "habr.com", "Base URL for replace, if \"\" no replace performed")

	flag.StringVar(&logLevel, "log_level", "error", "logging level, can be set to: panic, error, warning, info, debug.")
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

	// Инициализируем обработчик
	rotPRfunc := http.HandlerFunc(PRHandler)
	http.Handle("/", rotPRfunc)
	// Запускаем основной обработчик в отдельной горутине, что бы продолжить и отлавливать SIGTERM
	go func() {
		log.Debug("ok: http.Handle(/, PRHandler)")
		if errQr := http.ListenAndServe("127.0.0.1:8080", nil); errQr != nil {
			log.Error("Err: http.ListenAndServe(\"127.0.0.1:8080\") error: %s" + errQr.Error())
		}
	}()

	// Ждем сигнала завершения работы SIGTERM .
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)
	<-sigC
	log.Info("SIGTERM receided, exiting")
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
	for k, v := range r.Header {
		for val := range v {
			if k != "Connection" && k != "Keep-Alive" { //убираем заголовки подддержки удержания соединения
				CallReq.Header.Add(k, v[val])
				log.Debugf("ok: Request             Added Header | %s | %s \n", k, v[val])
			}
		}
	}
	CallReq.Header.Set("Connection", "close")
	CallReq.Header.Set("Keep-Alive", "timeout=0")

	// выполняем запрос.
	result, err := Cl.Do(CallReq)
	if err != nil {
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
	// если в ответе HTML, то обрабатываем
	if strings.Contains(w.Header().Get("Content-Type"), "text/html") {
		if w.Header().Get("Content-Encoding") == "gzip" { // содержимое запаковано
			// предварительно распаковываем
			Body, err = io.ReadAll(result.Body)
			if err != nil {
				log.Error("err: Error read from server reply Body: " + err.Error())
			}
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
			ResZParse, err := ParseRespBody(dataUnZip)
			if err != nil {
				log.Error("err: Error ParseRespBody: " + err.Error())
			}

			// тут пакуем обратно и пишем в ответ браузеру
			gw, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
			if err != nil {
				log.Error("err: Error Init new Gzip Writer: " + err.Error())
			}
			defer gw.Close()
			//log.Debugf("ok: Received  Replay: \n %s\n ", string(ResZParse))
			gw.Write(ResZParse)

		} else { // содержимое не запаковано, обрабатываем сразу.
			Body, _ = io.ReadAll(result.Body)
			log.Debug("ok: Контент не упакован, сразу его полностью читаем.")
			ResParse, _ := ParseRespBody(Body)
			//	log.Debugf("ok: Received  Replay: \n %s\n ", string(ResParse))
			w.Write(ResParse)
		}
	} else {
		// не HTML передаем сразу без обработки
		Body, err = io.ReadAll(result.Body)
		if err != nil {
			log.Error("err: Error read from server reply Body: " + err.Error())
		}
		log.Debugf("ok: Received  Replay: \n %s\n ", string(Body))
		w.Write(Body)
	}
}

// ParseRespBody - обработчик ответа сервера, сканирует HTML, ищет выводимый текст и слова в нем длинна которых = 6 символам,
// и вставляет после найденных слов символ "\u2122
func ParseRespBody(InText []byte) ([]byte, error) {
	var (
		Result      []byte
		ReadBuffer  *bytes.Buffer
		WriteBuffer *bytes.Buffer
		ResErr      error = nil
	)
	ReadBuffer = bytes.NewBuffer(InText)
	WriteBuffer = bytes.NewBuffer(make([]byte, len(InText)+40))

	// внутренние функции обработки потока.
	ReadTag := func() string {
		var b strings.Builder
		//	log.Debugf("ReadTag пока быстро вернулись")
		rr, _, err := ReadBuffer.ReadRune()
		if err != nil {
			log.Error("err: Error read new Rune from ReadBuffer: " + err.Error())
			ResErr = err
		}
		for rr != '>' {
			b.WriteRune(rr)
			WriteBuffer.WriteRune(rr)
			rr, _, err = ReadBuffer.ReadRune()
			if err != nil {
				log.Error("err: Error read new Rune from ReadBuffer: " + err.Error())
				ResErr = err
			}
		}
		WriteBuffer.WriteRune(rr)
		return b.String()
	}

	ReadScript := func() {
		rr, _, err := ReadBuffer.ReadRune()
		if err != nil {
			log.Error("err: Error read new Rune from ReadBuffer: " + err.Error())
			ResErr = err
		}
		for rr != '<' {
			WriteBuffer.WriteRune(rr)
			rr, _, err = ReadBuffer.ReadRune()
			if err != nil {
				log.Error("err: Error read new Rune from ReadBuffer: " + err.Error())
				ResErr = err
			}
		}
		WriteBuffer.WriteRune(rr)
		//log.Debugf("ReadScript '%s'", rr)
		ss := ReadTag()
		if !strings.Contains(ss, "script") && !strings.Contains(ss, "style") {
			log.Warning("err: <stcript> or <style> not ended with </stcript> or </style>")
		}

	}

	CountSymb := 0 // счетчик букв в слове
	for {
		r, _, err := ReadBuffer.ReadRune()
		if err != nil {
			break
		}
		switch r {

		case '<':
			// читаем дальше тег
			CountSymb = 0
			WriteBuffer.WriteRune(r)
			//		log.Debugf("нашли < %s\n", r)
			Tag := ReadTag()
			if strings.Contains(Tag, "script") || strings.Contains(Tag, "style") {
				ReadScript()
			}

		case ' ', ',', ';', ':', '-', '\n', '(', ')', '"', '.', '»', '«':
			//	нашли конец слова, если нужно вставляем символ
			if CountSymb == 6 {
				WriteBuffer.WriteRune('\u2122')
			}
			WriteBuffer.WriteRune(r)
			CountSymb = 0

		default:
			// обрабатываем как слово, увеличиваем счетчик
			CountSymb++
			WriteBuffer.WriteRune(r)
		}

	}

	Result = WriteBuffer.Bytes()

	return Result, ResErr
}
