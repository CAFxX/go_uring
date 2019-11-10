FROM golang:latest

WORKDIR /project
COPY iouring.go /project/iouring.go
RUN wget https://git.kernel.dk/cgit/liburing/snapshot/liburing-0.2.tar.gz
RUN tar -xf liburing-0.2.tar.gz

WORKDIR /project/liburing-0.2
RUN make
RUN make install

WORKDIR /project
RUN go test
