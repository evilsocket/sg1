TARGET=sg1
BUILD_FOLDER=build

all: sg1
	@echo "@ Done"
	@echo -n "\n"

sg1: build_folder
	@echo "@ Building $(TARGET) ..."
	@go build $(FLAGS) -o $(BUILD_FOLDER)/$(TARGET) main.go

test: build_file
	@go test ./... -v

build_folder:
	@mkdir -p $(BUILD_FOLDER)

clean:
	@rm -rf $(TARGET) $(BUILD_FOLDER)
