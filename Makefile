all:
	python sort.py 04.yaml 04.yaml
	python sort.py 50.yaml 50.yaml
	go run *.go > out.markdown
