FROM ubuntu

RUN apt-get update
RUN apt-get install -y openjdk-11-jdk
RUN apt-get install -y curl

WORKDIR /usr/local/lib
RUN curl -O http://www.antlr.org/download/antlr-4.7.1-complete.jar
ENV CLASSPATH=".:/usr/local/lib/antlr-4.7.1-complete.jar:$CLASSPATH"

WORKDIR /
ENTRYPOINT [ "java", "-Xmx500M", "-cp", "/usr/local/lib/antlr-4.7.1-complete.jar:$CLASSPATH", "org.antlr.v4.Tool" ]