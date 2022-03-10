# TMProxy
Прокси принимающий http:// запросы на порт 8080, и перенаправляющий их на внешний сервер.
Ответы сервера обрабатываются, в отображаемом тексте после слов длинной 6 символов добавляется знак u\2122

Использование:
make build        - создаст выполняемый файл в каталоге cmd, в системе должен быть установлен GO.
make run          - запуск на выполнение 
make docker_build - создание Docker образа, наличие на системе GO не требуется, сборка ведется внутри промежуточного образа.
make docker_run   - запуск созданного образа, по умолчанию пробрасывается порт 8080.

Аргументы командной строки:
-BaseURL string - если если переданное значение параметра не "" то оно дуте заменять HOST:PORT в запросе к серверу. По умолчанию habr.com.
-LogLevel - уровень логирования, принимаются значения panic, error, warning, info, debug. По умолчанию - error