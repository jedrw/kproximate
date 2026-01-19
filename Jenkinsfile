pipeline {
    agent any

    environment {
        GCP_REGION = "${env.GCP_REGION ?: 'us-central1'}"
        GCP_PROJECT_ID = "${env.GCP_PROJECT_ID ?: 'customarypaas'}"
        REGISTRY = "${GCP_REGION}-docker.pkg.dev"
        IMAGE_BASE = "${REGISTRY}/${GCP_PROJECT_ID}/kproximate"
        VERSION = "${env.GIT_TAG_NAME ?: "0.0.0-${env.BUILD_NUMBER}"}"
    }

    stages {
        stage('Manual Approval') {
            when {
                branch 'main'
            }
            steps {
                script {
                    def userInput = input(
                        message: "Approve build for ${env.BRANCH_NAME}?",
                        ok: 'Build',
                        parameters: [
                            string(name: 'Reason', defaultValue: '', description: 'Reason for approval')
                        ]
                    )
                    echo "Approval Reason: ${userInput}"
                }
            }
        }

        stage('Build Images') {
            parallel {
                stage('Build Controller') {
                    steps {
                        container('dind') {
                            script {
                                docker.withRegistry("https://${REGISTRY}") {
                                    sh "docker-credential-gcr configure-docker --registries=${REGISTRY}"
                                    sh """
                                        docker build -f Dockerfile \
                                            --build-arg COMPONENT=controller \
                                            --build-arg TARGETARCH=amd64 \
                                            -t ${IMAGE_BASE}-controller:${VERSION} \
                                            -t ${IMAGE_BASE}-controller:latest \
                                            .
                                    """
                                    sh "docker push ${IMAGE_BASE}-controller:${VERSION}"
                                    sh "docker push ${IMAGE_BASE}-controller:latest"
                                }
                            }
                        }
                    }
                }

                stage('Build Worker') {
                    steps {
                        container('dind') {
                            script {
                                docker.withRegistry("https://${REGISTRY}") {
                                    sh """
                                        docker build -f Dockerfile \
                                            --build-arg COMPONENT=worker \
                                            --build-arg TARGETARCH=amd64 \
                                            -t ${IMAGE_BASE}-worker:${VERSION} \
                                            -t ${IMAGE_BASE}-worker:latest \
                                            .
                                    """
                                    sh "docker push ${IMAGE_BASE}-worker:${VERSION}"
                                    sh "docker push ${IMAGE_BASE}-worker:latest"
                                }
                            }
                        }
                    }
                }
            }
        }

        stage('Tag Release') {
            when {
                branch 'main'
            }
            steps {
                script {
                    sh """
                        git tag -a v${VERSION} -m "Release v${VERSION}" || true
                        git push origin v${VERSION} || true
                    """
                }
            }
        }
    }

    post {
        success {
            echo "Images built successfully:"
            echo "  Controller: ${IMAGE_BASE}-controller:${VERSION}"
            echo "  Worker: ${IMAGE_BASE}-worker:${VERSION}"
        }
        failure {
            echo "Build failed"
        }
    }
}
