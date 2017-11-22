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

clean:
	@rm -rf $(TARGET) $(BUILD_FILE)
