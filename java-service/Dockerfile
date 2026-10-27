# ---- build stage: JDK + Maven, dibuang setelah build ----
FROM maven:3.9-eclipse-temurin-21 AS builder
WORKDIR /build
# Cache dependency: layer ini hanya dibangun ulang bila pom.xml berubah.
COPY pom.xml ./
RUN mvn -B -q dependency:go-offline
COPY src ./src
RUN mvn -B -q package -DskipTests

# ---- runtime stage: hanya JRE (tanpa compiler javac) + fat jar ----
FROM eclipse-temurin:21-jre-alpine
WORKDIR /app
RUN addgroup -S app && adduser -S app -G app
COPY --from=builder /build/target/app.jar app.jar
USER app
# JVM membaca batas memori container. Default heap = 25% RAM container; dinaikkan ke 75%.
# Catatan: dengan RAM < 1792 MB JVM otomatis memilih SerialGC (bukan G1).
ENV JAVA_TOOL_OPTIONS="-XX:MaxRAMPercentage=75 -XX:+ExitOnOutOfMemoryError"
EXPOSE 8080
ENTRYPOINT ["java", "-jar", "/app/app.jar"]
