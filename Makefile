TARGET=sg1
BUILD_DATE=`date +%Y-%m-%d\ %H:%M`
BUILD_COMMIT=$(shell git rev-parse HEAD)
BUILD_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
BUILD_FOLDER=build
BUILD_FILE=sg1/build.go

all: sg1
	@echo "@ Done"
	@echo -n "\n"

sg1: build_folder
	@echo "@ Building $(TARGET) ..."
	@go build $(FLAGS) -o $(BUILD_FOLDER)/$(TARGET) main.go

build_folder:
	@mkdir -p $(BUILD_FOLDER)

build_file: 
	@rm -f $(BUILD_FILE)
	@echo "package sg1" > $(BUILD_FILE)
	@echo "const (" >> $(BUILD_FILE)
	@echo "  APP_BUILD_DATE = \"$(BUILD_DATE)\"" >> $(BUILD_FILE)
	@echo "  APP_BUILD_BRANCH = \"$(BUILD_BRANCH)\"" >> $(BUILD_FILE)
	@echo "  APP_BUILD_COMMIT = \"$(BUILD_COMMIT)\"" >> $(BUILD_FILE)
	@echo ")" >> $(BUILD_FILE)

clean:
	@rm -rf $(TARGET) $(BUILD_FILE) $(BUILD_FOLDER)
