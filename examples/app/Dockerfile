FROM --platform=linux/amd64 openjdk:11 
WORKDIR /app
COPY PrimeMain.java /app
RUN javac -g PrimeMain.java
RUN jar --create --file /app/PrimeMain.jar --main-class PrimeMain PrimeMain.class
RUN useradd -u 1234 appuser
USER appuser
ENTRYPOINT ["java", "-jar", "/app/PrimeMain.jar"]
