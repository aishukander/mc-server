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
if ! command -v java &> /dev/null; then
    echo "Installing Java ${JAVA_VERSION} via microdnf"

    case "${JAVA_VERSION}" in
        8)  pkg="java-1.8.0-openjdk";;
        16) pkg="java-16-openjdk";;
        17) pkg="java-17-openjdk";;
        21) pkg="java-21-openjdk";;
        *)  echo "Unsupported JAVA_VERSION=${JAVA_VERSION}, default to 1.8"; pkg="java-1.8.0-openjdk-headless";;
    esac

    microdnf install -y "${pkg}"
fi

cd /project/server

# Create eula.txt
if [ ! -f "eula.txt" ]; then
    echo "# Created with docker" > eula.txt
    echo "$(LC_TIME=en_US.UTF-8 date '+# %a %b %d %I:%M:%S %p %Z %Y')" >> eula.txt
    echo "eula=true" >> eula.txt
fi

case "$Type" in
    "paper")
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
        ;;
    
    "neoforge")
        echo "-Xms${Min_Ram} -Xmx${Max_Ram}" > user_jvm_args.txt

        if [ -n "$NEO_VERSION_OVERRIDE" ] && [ ! -d "./libraries/net/neoforged/neoforge/$NEO_VERSION_OVERRIDE" ]; then
            echo "Changing the neoforge version..."
            rm -rf libraries logs
            if [ -f "run.sh" ]; then
                rm run.sh
            fi
        fi

        if [ ! -f "run.sh" ]; then
            version_filter="${MINECRAFT_VERSION#1.}"
            api_url="https://maven.neoforged.net/releases/net/neoforged/neoforge"
            if [ -n "$NEO_VERSION_OVERRIDE" ]; then
                neo_version="$NEO_VERSION_OVERRIDE"
            else
                neo_version=$(wget -qO- "$api_url/maven-metadata.xml" | \
                    xmllint --xpath "string(//metadata/versioning/versions/version[contains(text(),'${version_filter}')][last()])" -)
                if [ -z "$neo_version" ]; then
                    echo "Unsupported versions of Minecraft"
                    exit 0
                fi
            fi

            wget -O "neoforge-${MINECRAFT_VERSION}.jar" "$api_url/$neo_version/neoforge-$neo_version-installer.jar"
            java -jar "neoforge-${MINECRAFT_VERSION}.jar"
            rm "neoforge-${MINECRAFT_VERSION}.jar"
        fi

        ./run.sh
        ;;
    
    *)
        jar_name="${Type}-${MINECRAFT_VERSION}.jar"
        if [ ! -f "$jar_name" ]; then
            download_url="https://mcutils.com/api/server-jars/${Type}/${MINECRAFT_VERSION}/download"
            if ! wget -q -O "$jar_name" "$download_url"; then
                echo "Unsupported versions of Minecraft"
                exit 0
            fi
        fi
        # Run server
        java -Xms${Min_Ram} -Xmx${Max_Ram} -jar "$jar_name" nogui
        ;;
esac