pipeline {
	agent {
		docker {
			image 'golang:1.10.3'
			args "-v /etc/passwd:/etc/passwd -v /home/jenkins/.ssh:/home/jenkins/.ssh -v /var/run/docker.sock:/var/run/docker.sock:rw --group-add 999"
		}
	}
	options {
		disableConcurrentBuilds()
		buildDiscarder(logRotator(numToKeepStr: "20"))
		skipStagesAfterUnstable()
		timestamps()
	}
	environment {
		HIPCHAT = "4698225" // EMT-Monitoring
		ARTIFACTORY_CREDS = credentials('jenkins-jira')
		SPINNAKER_CREDS = credentials('spinnaker-emt-auth')
		VERSION = "${com.kiwigrid.ecs.jenkinsci.PipelineHelper.isMasterOrRelease(BRANCH_NAME) ? com.kiwigrid.ecs.jenkinsci.PipelineHelper.generateVersionFromHistory(this, BRANCH_NAME, '0') : '0.0.' + BUILD_NUMBER + '-' + BRANCH_NAME.replaceAll('/', '-') + '-SNAPSHOT'}"
		IMG="docker.kiwigrid.com/emt/spinnaker-manager:${VERSION}"
	}
	stages {
		stage("Init") {
			steps {
				sh "echo 'version=${VERSION}' > version.properties"
				script {
					currentBuild.description = env.VERSION
				}
			}
		}
		stage("Build") {
			steps {
				sh "curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh"
				sh "dep ensure"
				sh "make"
				sh "make docker-build"
			}
		}
		stage("Publish") {
			steps {
                sh """
                    export DOCKER_CONFIG=\$WORKSPACE
                    docker login --username=${ARTIFACTORY_CREDS_USR} --password=${ARTIFACTORY_CREDS_PSW} docker.kiwigrid.com
                    make docker-push
                """
			}
		}
		stage("Git Tag") {
			when {
				branch "master"
			}
			steps {
				sh "git tag ${VERSION}"
				sh "git push origin ${VERSION}"
			}
		}
	}
	post {
		success {
			hipchatSend(color: 'GREEN', message: "Gcp Service Account Controller: build success on ${BRANCH_NAME} (<a href=\"${BUILD_URL})\">Build #${BUILD_NUMBER}</a>)", room: "${HIPCHAT}", notify: false, failOnError: true)
		}
		failure {
			hipchatSend(color: 'RED', message: "Gcp Service Account Controller: build failed on ${BRANCH_NAME} (<a href=\"${BUILD_URL})\">Build #${BUILD_NUMBER}</a>)", room: "${HIPCHAT}", notify: false, failOnError: true)
		}
	}
}
