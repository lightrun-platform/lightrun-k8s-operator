FROM openjdk:11 
WORKDIR /app
COPY PrimeMain.java /app
RUN javac -g PrimeMain.java
RUN jar --create --file /app/PrimeMain.jar --main-class PrimeMain PrimeMain.class
USER 1234
ENTRYPOINT ["java", "-jar", "/app/PrimeMain.jar"]
