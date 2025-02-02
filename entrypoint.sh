#!/bin/bash

# Detect the required java version
if [ -n "$JAVA_VERSION_OVERRIDE" ]; then
    JAVA_VERSION="$JAVA_VERSION_OVERRIDE"
else
    version_ge() {
        printf '%s\n' "$2" "$1" | sort -V | head -n1 | grep -q "$2"
    }

    JAVA_VERSION="8"

    if [[ -n "$MINECRAFT_VERSION" ]]; then
        if version_ge "$MINECRAFT_VERSION" "1.20.5"; then
            JAVA_VERSION="21"
        elif version_ge "$MINECRAFT_VERSION" "1.18"; then
            JAVA_VERSION="17"
        elif version_ge "$MINECRAFT_VERSION" "1.17"; then
            JAVA_VERSION="16"
        elif version_ge "$MINECRAFT_VERSION" "1.12"; then
            JAVA_VERSION="8"
        fi
    fi
fi

# Install java
if [ ! -d "/project/java/jdk${JAVA_VERSION}" ]; then
    echo "start install java ${JAVA_VERSION}"
    LATEST_URL=$(wget -qO- "https://api.github.com/repos/adoptium/temurin${JAVA_VERSION}-binaries/releases/latest" \
        | grep "browser_download_url" \
        | grep "jdk_x64_linux" \
        | head -n 1 \
        | cut -d '"' -f 4)
    wget -O jdk${JAVA_VERSION}.tar.gz "${LATEST_URL}"
    tar -xzf jdk${JAVA_VERSION}.tar.gz

    jdk_dir=$(tar -tf jdk${JAVA_VERSION}.tar.gz | head -1 | cut -f1 -d"/")
    mkdir -p /project/java
    echo "Moving ${jdk_dir} to /project/java"
    mv "${jdk_dir}" /project/java/jdk${JAVA_VERSION}
    rm jdk${JAVA_VERSION}.tar.gz
fi

export PATH=/project/java/jdk${JAVA_VERSION}/bin:$PATH

cd /project/server

# Create eula.txt
if [ ! -f "eula.txt" ]; then
    echo "# Created with docker" > eula.txt
    echo "$(LC_TIME=en_US.UTF-8 date '+# %a %b %d %I:%M:%S %p %Z %Y')" >> eula.txt
    echo "eula=true" >> eula.txt
fi

# paper
if [ "$Type" = "paper" ]; then
    if [ ! -f "paper-${MINECRAFT_VERSION}.jar" ]; then
        api_url="https://api.papermc.io/v2/projects/paper/versions/${MINECRAFT_VERSION}/builds"
        latest_build=$(wget -qO- "$api_url" | jq '.builds[-1].build')
        if [ "$latest_build" = "null" ] || [ -z "$latest_build" ]; then
            echo "Unsupported versions of Minecraft"
            exit 0
        fi
        download_url="https://api.papermc.io/v2/projects/paper/versions/${MINECRAFT_VERSION}/builds/${latest_build}/downloads/paper-${MINECRAFT_VERSION}-${latest_build}.jar"
        wget -O "paper-${MINECRAFT_VERSION}.jar" "$download_url"
    fi

    # Run server
    java -Xms${Min_Ram} -Xmx${Max_Ram} -jar "paper-${MINECRAFT_VERSION}.jar" nogui
fi

# neoforge
if [ "$Type" = "neoforge" ]; then
    echo "-Xms${Min_Ram} -Xmx${Max_Ram}" > user_jvm_args.txt

    if [ ! -f "run.sh" ]; then
        version_filter="${MINECRAFT_VERSION#1.}"
        api_url="https://maven.neoforged.net/releases/net/neoforged/neoforge"
        latest_version=$(wget -qO- "$api_url/maven-metadata.xml" | xmllint --xpath "string(//metadata/versioning/versions/version[contains(text(),'${version_filter}')][last()])" -)
        if [ -z "$latest_version" ]; then
            echo "Unsupported versions of Minecraft"
            exit 0
        fi
        wget -O "neoforge-${MINECRAFT_VERSION}.jar" "$api_url/$latest_version/neoforge-$latest_version-installer.jar"
        java -jar "neoforge-${MINECRAFT_VERSION}.jar"
        rm "neoforge-${MINECRAFT_VERSION}.jar"
    fi

    ./run.sh
fi

# fabric
if [ "$Type" = "fabric" ]; then
    if [ ! -f "fabric-${MINECRAFT_VERSION}.jar" ]; then
        download_url="https://mcutils.com/api/server-jars/fabric/${MINECRAFT_VERSION}/download"
        wget -O "fabric-${MINECRAFT_VERSION}.jar" "$download_url" || {
            echo "Unsupported versions of Minecraft"
            exit 0
        }
    fi
    # Run server
    java -Xms${Min_Ram} -Xmx${Max_Ram} -jar "fabric-${MINECRAFT_VERSION}.jar" nogui
fi