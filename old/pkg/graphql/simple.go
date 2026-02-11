package graphql

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// SimpleGraphQLServer provides a basic GraphQL implementation
type SimpleGraphQLServer struct {
	config *GraphQLConfig
	schema string
}

// NewSimpleGraphQLServer creates a new simple GraphQL server
func NewSimpleGraphQLServer(config *GraphQLConfig, schema string) *SimpleGraphQLServer {
	if config == nil {
		config = DefaultGraphQLConfig()
	}

	return &SimpleGraphQLServer{
		config: config,
		schema: schema,
	}
}

// RegisterRoutes registers GraphQL routes with Fiber
func (sgs *SimpleGraphQLServer) RegisterRoutes(app *fiber.App) {
	// GraphQL endpoint
	app.All(sgs.config.Path, sgs.handleGraphQL)

	// GraphQL playground (if enabled)
	if sgs.config.Playground {
		playgroundPath := sgs.config.Path + "/playground"
		app.Get(playgroundPath, sgs.handlePlayground)
	}
}

// handleGraphQL handles GraphQL requests
func (sgs *SimpleGraphQLServer) handleGraphQL(c *fiber.Ctx) error {
	// Set CORS headers
	c.Set("Access-Control-Allow-Origin", "*")
	c.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if c.Method() == "OPTIONS" {
		return c.SendStatus(200)
	}

	// Handle GET requests (query in URL)
	if c.Method() == "GET" {
		query := c.Query("query")
		if query == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "query parameter is required for GET requests",
			})
		}

		variables := c.Query("variables")
		var variablesMap map[string]interface{}
		if variables != "" {
			if err := json.Unmarshal([]byte(variables), &variablesMap); err != nil {
				return c.Status(400).JSON(fiber.Map{
					"error": "invalid variables JSON",
				})
			}
		}

		return sgs.executeQuery(c, query, variablesMap)
	}

	// Handle POST requests
	if c.Method() == "POST" {
		var requestBody struct {
			Query         string                 `json:"query"`
			Variables     map[string]interface{} `json:"variables"`
			OperationName string                 `json:"operationName"`
		}

		if err := c.BodyParser(&requestBody); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid request body",
			})
		}

		if requestBody.Query == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "query is required",
			})
		}

		return sgs.executeQuery(c, requestBody.Query, requestBody.Variables)
	}

	return c.Status(405).JSON(fiber.Map{
		"error": "method not allowed",
	})
}

// executeQuery executes a GraphQL query
func (sgs *SimpleGraphQLServer) executeQuery(c *fiber.Ctx, query string, variables map[string]interface{}) error {
	// This is a simplified implementation
	// In a real implementation, you would parse and execute the GraphQL query

	// For now, return a mock response based on the query
	response := sgs.generateMockResponse(query, variables)

	return c.Status(200).JSON(response)
}

// generateMockResponse generates a mock response based on the query
func (sgs *SimpleGraphQLServer) generateMockResponse(query string, variables map[string]interface{}) map[string]interface{} {
	query = strings.TrimSpace(query)

	// Simple query detection
	if strings.Contains(query, "users") {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"users": []map[string]interface{}{
					{
						"id":    "1",
						"name":  "John Doe",
						"email": "john@example.com",
					},
					{
						"id":    "2",
						"name":  "Jane Smith",
						"email": "jane@example.com",
					},
				},
			},
		}
	}

	if strings.Contains(query, "products") {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"products": []map[string]interface{}{
					{
						"id":          "1",
						"name":        "Laptop",
						"description": "High-performance laptop",
						"price":       999.99,
						"in_stock":    true,
					},
					{
						"id":          "2",
						"name":        "Mouse",
						"description": "Wireless mouse",
						"price":       29.99,
						"in_stock":    true,
					},
				},
			},
		}
	}

	// Default response
	return map[string]interface{}{
		"data": map[string]interface{}{
			"message": "GraphQL query executed successfully",
			"query":   query,
		},
	}
}

// handlePlayground handles GraphQL playground requests
func (sgs *SimpleGraphQLServer) handlePlayground(c *fiber.Ctx) error {
	// Simple HTML playground
	html := sgs.generatePlaygroundHTML()

	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

// generatePlaygroundHTML generates a simple GraphQL playground
func (sgs *SimpleGraphQLServer) generatePlaygroundHTML() string {
	return `<!DOCTYPE html>
<html>
<head>
    <title>GraphQL Playground</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background-color: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; text-align: center; margin-bottom: 30px; }
        .query-section { margin-bottom: 30px; }
        .query-section h3 { color: #555; margin-bottom: 10px; }
        textarea { width: 100%; height: 200px; padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-family: monospace; }
        button { background-color: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; margin-right: 10px; }
        button:hover { background-color: #0056b3; }
        .result { margin-top: 20px; padding: 15px; background-color: #f8f9fa; border: 1px solid #e9ecef; border-radius: 4px; }
        .result pre { margin: 0; white-space: pre-wrap; }
        .error { color: #dc3545; }
        .success { color: #28a745; }
    </style>
</head>
<body>
    <div class="container">
        <h1>GraphQL Playground</h1>
        
        <div class="query-section">
            <h3>Query</h3>
            <textarea id="query" placeholder="Enter your GraphQL query here...">{
  users {
    id
    name
    email
  }
}</textarea>
        </div>
        
        <div class="query-section">
            <h3>Variables (optional)</h3>
            <textarea id="variables" placeholder='{"key": "value"}'></textarea>
        </div>
        
        <button onclick="executeQuery()">Execute Query</button>
        <button onclick="clearResult()">Clear</button>
        
        <div id="result" class="result" style="display: none;">
            <h3>Result</h3>
            <pre id="resultContent"></pre>
        </div>
    </div>

    <script>
        async function executeQuery() {
            const query = document.getElementById('query').value;
            const variablesText = document.getElementById('variables').value;
            const resultDiv = document.getElementById('result');
            const resultContent = document.getElementById('resultContent');
            
            if (!query.trim()) {
                showError('Please enter a query');
                return;
            }
            
            let variables = {};
            if (variablesText.trim()) {
                try {
                    variables = JSON.parse(variablesText);
                } catch (e) {
                    showError('Invalid JSON in variables: ' + e.message);
                    return;
                }
            }
            
            try {
                const response = await fetch('` + sgs.config.Path + `', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        query: query,
                        variables: variables
                    })
                });
                
                const data = await response.json();
                
                if (data.errors) {
                    showError('GraphQL Errors: ' + JSON.stringify(data.errors, null, 2));
                } else {
                    showSuccess(JSON.stringify(data, null, 2));
                }
            } catch (error) {
                showError('Network Error: ' + error.message);
            }
        }
        
        function showError(message) {
            const resultDiv = document.getElementById('result');
            const resultContent = document.getElementById('resultContent');
            
            resultDiv.style.display = 'block';
            resultContent.innerHTML = '<span class="error">' + message + '</span>';
        }
        
        function showSuccess(message) {
            const resultDiv = document.getElementById('result');
            const resultContent = document.getElementById('resultContent');
            
            resultDiv.style.display = 'block';
            resultContent.innerHTML = '<span class="success">' + message + '</span>';
        }
        
        function clearResult() {
            document.getElementById('result').style.display = 'none';
        }
    </script>
</body>
</html>`
}

// GetConfig returns the GraphQL configuration
func (sgs *SimpleGraphQLServer) GetConfig() *GraphQLConfig {
	return sgs.config
}

// GetSchema returns the GraphQL schema
func (sgs *SimpleGraphQLServer) GetSchema() string {
	return sgs.schema
}

// SetSchema sets the GraphQL schema
func (sgs *SimpleGraphQLServer) SetSchema(schema string) {
	sgs.schema = schema
}
