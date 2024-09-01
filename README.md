# Goland JSON Sync Service



This a backend service written in Golang using Gin and WebSocket to keep JSON data synchronized between clients. It utilizes Redis for persistence, ensuring quick response times. The service allows clients to upload JSON data, receive updates from other clients via WebSocket, and query the entire thing, ideal for applications that require real-time synchronization and data consistency across multiple clients.


## Technologies Used

- **Gin**: A high-performance web framework for Go.
- **WebSocket**: For real-time bidirectional communication.
- **Redis**: In-memory database for fast data storage and retrieval.
- **Golang**: A statically typed, compiled programming language designed at Google. 


## Installation

1. **Clone the Repository**
    ```bash
    git clone https://github.com/Vadim-Karpenko/golang-json-sync-service.git
    cd golang-json-sync-service
    ```
2. **Install Dependencies**

    Ensure you have Go installed. Then, use the following command to install the necessary Go packages:
    ```bash
   go mod tidy
   ```

3. **Run Redis**

   Make sure you have Redis installed and running. You can use Docker to run Redis:
    ```bash
   docker run -d -p 6379:6379 redis
    ```

4. **Build and Run the Server**

   Run the server using:
    ```bash
   go run
    ```

    The server will start on `http://localhost:8080`.

5. **Build**

    If you want to build the executable, use:
    ```bash
    go build
    ```
## API Endpoints

### Upload JSON

- **Endpoint**: `POST /upload`
- **Request Body**: Any JSON data
- **Response**: `{ "uuid": "unique-identifier" }`

Please note that the default timeout for the JSON data is 30 days after the last update. After that, it will be deleted from the Redis database. Edit the source code if you want to change this behavior.

### WebSocket Connection

- **Endpoint**: `ws://localhost:8080/ws/:uuid` (use wss:// if in production!)
- **Request Body**: `{ "path": "character.items.1.name", "value": "new value" }`

If you have a list of items, you can use the index to update the specific item. The path should be in the format of `character.items.1.name` to update the name of the second item in the list.

### Retrieve JSON

- **Endpoint**: `GET /json/:uuid`
- **Response**: JSON data

## Testing

To run the tests for this project, execute:
```bash
go test
```
## Example

1. **Upload JSON Data**
    ```bash
   curl -X POST http://localhost:8080/upload -d '{"character": {"name": "Frodo", "age": 50}}' -H "Content-Type: application/json"
    ```

2. **Connect via WebSocket**

   Use a WebSocket client to connect to `ws://localhost:8080/ws/{uuid}` and send updates.

   ```json
    {"path": "character.age", "value": 51}
   ```

3. **Retrieve Updated JSON**
    
    Other connected users will receive the exact message you sent for an update. You'll need to use it to update the locally stored data in your application.

    If it's too complicated in your case, you can just retrieve the whole thing by using:
    ```bash
    curl http://localhost:8080/json/{uuid}
    ```
    Also, if your user just got connected, it would be wise to use this API during the init phase.

## Contributing

Contributions are welcome! Please submit issues and pull requests on GitHub.

## Contact

For any questions or support, please contact [vadim@karpenko.work](mailto:vadim@karpenko.work).
