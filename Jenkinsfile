pipeline {
    agent any

    environment {
        GCP_REGION = "${env.GCP_REGION ?: 'us-central1'}"
        GCP_PROJECT_ID = "${env.GCP_PROJECT_ID ?: 'customarypaas'}"
        REGISTRY = "${GCP_REGION}-docker.pkg.dev"
        IMAGE_NAME = "${REGISTRY}/${GCP_PROJECT_ID}/kproximate"
    }

    stages {
        stage('Manual Approval') {
            when {
                expression { 
                    return (env.BRANCH_NAME == 'main' || env.GIT_BRANCH == 'origin/main')
                }
            }
            steps {
                script {
                    def userInput = input(
                        message: "Approve build for ${env.BRANCH_NAME ?: env.GIT_BRANCH}?",
                        ok: 'Build',
                        parameters: [
                            string(name: 'Reason', defaultValue: '', description: 'Reason for approval')
                        ]
                    )
                    echo "Approval Reason: ${userInput}"
                }
            }
        }

        stage('Build Controller Image') {
            steps {
                container('dind') {
                    echo 'Building controller Docker image'
                    script {
                        docker.withRegistry("https://${REGISTRY}") {
                            sh "docker-credential-gcr configure-docker --registries=${REGISTRY}"
                            sh """
                                docker build . \
                                    --build-arg COMPONENT=controller \
                                    --build-arg TARGETARCH=amd64 \
                                    -t ${IMAGE_NAME}/controller:${BUILD_NUMBER} \
                                    -t ${IMAGE_NAME}/controller:latest
                            """
                            sh "docker push ${IMAGE_NAME}/controller:${BUILD_NUMBER}"
                            sh "docker push ${IMAGE_NAME}/controller:latest"
                        }
                    }
                }
            }
        }

        stage('Build Worker Image') {
            steps {
                container('dind') {
                    echo 'Building worker Docker image'
                    script {
                        docker.withRegistry("https://${REGISTRY}") {
                            sh """
                                docker build . \
                                    --build-arg COMPONENT=worker \
                                    --build-arg TARGETARCH=amd64 \
                                    -t ${IMAGE_NAME}/worker:${BUILD_NUMBER} \
                                    -t ${IMAGE_NAME}/worker:latest
                            """
                            sh "docker push ${IMAGE_NAME}/worker:${BUILD_NUMBER}"
                            sh "docker push ${IMAGE_NAME}/worker:latest"
                        }
                    }
                }
            }
        }

        stage('Tag Git Commit') {
            when {
                expression { 
                    return (env.BRANCH_NAME == 'main' || env.GIT_BRANCH == 'origin/main')
                }
            }
            steps {
                script {
                    sh """
                        git tag -a v${BUILD_NUMBER} -m "Release v${BUILD_NUMBER}"
                        git push origin v${BUILD_NUMBER}
                    """
                }
            }
        }
    }

    post {
        success {
            echo "Images built successfully:"
            echo "  Controller: ${IMAGE_NAME}/controller:${BUILD_NUMBER}"
            echo "  Worker: ${IMAGE_NAME}/worker:${BUILD_NUMBER}"
        }
        failure {
            echo "Build failed"
        }
    }
}
