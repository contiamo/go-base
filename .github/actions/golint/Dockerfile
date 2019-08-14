FROM golang:1.12

LABEL repository="https://github.com/contiamo/go-base"
LABEL homepage="https://github.com/contiamo/go-base"
LABEL maintainer="Platform Team <developers@contiamo.com>"

LABEL "com.github.actions.name"="Go Lint Action"
LABEL "com.github.actions.description"="Wraps the golint command."
LABEL "com.github.actions.icon"="terminal"
LABEL "com.github.actions.color"="blue"

RUN go get -u golang.org/x/lint/golint

ENTRYPOINT ["golint -set_exit_status ./..."]
