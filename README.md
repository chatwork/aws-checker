# aws-checker

`aws-checker` is a long-running application that checks accessibility to various AWS services.

It exposes metrics compliant with the Prometheus exposition format so that you can monitor
if the checker, and hence your infrastructure, the platform running it, and the settings provided to the checker, is able to access the AWS services like:

- DynamoDB
- S3
- SQS

## Running locally

1. Grab the latest release for your os/arch from [releases](https://github.com/chatwork/aws-checker/releases).
2. Extract the tarball:
    ```sh
    tar -xzf aws-checker_<version>_<os>_<arch>.tar.gz
    ```
3. Run the application:
    ```sh
    ./aws-checker
    ```
4. The application will start and expose metrics at `http://localhost:8080/metrics`.

You can then use Prometheus or any other compatible monitoring tool to scrape the metrics from this endpoint.

## Run via docker

We publish the container images at https://github.com/chatwork/aws-checker/pkgs/container/aws-checker.

Use it for running the aws-checker within a container locally:

```
docker run -p 8080:8080 ghcr.io/chatwork/aws-checker:canary
```

Or in a Kubernetes pod.

## Running locally for development

To run `aws-checker` locally, follow these steps:

1. Ensure you have Go installed on your machine.
2. Clone the repository and navigate to the project directory.
3. Build the project:
    ```sh
    go build -o aws-checker
    ```
4. Run the application:
    ```sh
    ./aws-checker
    ```

The application will start and expose metrics at `http://localhost:8080/metrics`.

## Contributing

We welcome contributions to `aws-checker`!

Before submitting a pull request, we appreciate you to Write an issue describing the feature or bugfix you plan to work on and discuss it with the maintainers.

To contribute code changes, follow these steps:

1. Fork the repository on GitHub.
2. Clone your forked repository to your local machine:
    ```sh
    git clone https://github.com/<your-username>/aws-checker.git
    ```
3. Create a new branch for your feature or bugfix:
    ```sh
    git checkout -b my-feature-branch
    ```
4. Read the [LICENSE](LICENSE) file to understand the project's licensing terms.
5. Make your changes and commit them with descriptive commit messages:
    ```sh
    git add .
    git commit -m "Description of your changes"
    ```
6. Push your changes to your forked repository:
    ```sh
    git push origin my-feature-branch
    ```
7. Open a pull request on the main repository and describe your changes.

Please ensure your code follows the project's coding standards and includes appropriate tests.
