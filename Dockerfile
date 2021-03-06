# Creates Spring Boot Fat Jar
FROM openjdk:14-alpine
LABEL maintainer="fduchardt"
COPY . .
RUN ./gradlew bootJar
RUN mv ./build/libs/*.jar ./build/libs/k8s-hide-env.jar

# Debian container with kubectl and Java
FROM openjdk:14-alpine
LABEL maintainer="fduchardt"
RUN apk add --update openssl && \
    rm -rf /var/cache/apk/*
COPY --from=0 /build/libs/k8s-hide-env.jar ./
CMD java -jar k8s-hide-env.jar