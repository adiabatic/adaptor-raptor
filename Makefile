all:
	python sort.py > 51.yaml
	mv 51.yaml 50.yaml
	go run *.go > out.markdown
