# Creates Spring Boot Fat Jar
FROM openjdk:11
LABEL maintainer="fduchardt"
COPY . .
RUN ./gradlew bootJar
RUN mv ./build/libs/*.jar ./build/libs/k8s-hide-env.jar

# Debian container with kubectl and Java
FROM debian:buster
RUN apt update && \
      apt install -y curl && \
      curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl && \
      chmod +x ./kubectl && \
      mv ./kubectl /usr/local/bin/kubectl
RUN apt install -y openjdk-11-jre
LABEL maintainer="fduchardt"
COPY --from=0 /build/libs/k8s-hide-env.jar ./
CMD java -jar k8s-hide-env.jar