TARGET=sg1
BUILD_DATE=`date +%Y-%m-%d\ %H:%M`
BUILD_FILE=build.go

all: build
	@echo "@ Done"
	@echo -n "\n"

build: build_file
	@echo "@ Building ..."
	@go build $(FLAGS) -o $(TARGET) .

build_file: 
	@rm -f $(BUILD_FILE)
	@echo "package main" > $(BUILD_FILE)
	@echo "const (" >> $(BUILD_FILE)
	@echo "  APP_BUILD_DATE = \"$(BUILD_DATE)\"" >> $(BUILD_FILE)
	@echo ")" >> $(BUILD_FILE)

certs:
	@rm -rf certs
	@mkdir certs
	@echo "@ Generating server certificate ..."
	@openssl req -new -nodes -x509 -out certs/server.pem -keyout certs/server.key -days 3650 -subj "/C=DE/ST=NRW/L=Earth/O=Random Company/OU=IT/CN=www.random.com/emailAddress=jon@doe.com"
	@echo "@ Generating client certificate ..."
	@openssl req -new -nodes -x509 -out certs/client.pem -keyout certs/client.key -days 3650 -subj "/C=DE/ST=NRW/L=Earth/O=Random Company/OU=IT/CN=www.random.com/emailAddress=jon@doe.com"

clean:
	@rm -rf $(TARGET) $(BUILD_FILE)
