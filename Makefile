all: dictionaries
	go build -o adaptor-raptor raptor.go replacer.go entrysorter.go
	./adaptor-raptor > out.markdown

dictionaries: 04.yaml 50.yaml
	python sort.py 04.yaml 04.yaml
	python sort.py 50.yaml 50.yaml

