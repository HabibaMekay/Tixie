# Test-UserService.ps1

$baseUrl = "http://localhost:8081/v1"


$user = @{
    username = "testuser"
    email = "testuser@example.com"
    password = "TestPass123"
}
$userJson = $user | ConvertTo-Json

Write-Host "Creating user..."
$response = Invoke-RestMethod -Uri $baseUrl -Method Post -Body $userJson -ContentType "application/json"
Write-Host "User creation status: $($response.StatusCode)" -ForegroundColor Green

Start-Sleep -Seconds 1

Write-Host "Getting all users..."
$response = Invoke-RestMethod -Uri $baseUrl -Method Get
$response | ConvertTo-Json -Depth 3

# Get user ID 
$userId = $response[0].id

Write-Host "Getting user by ID $userId..."
$response = Invoke-RestMethod -Uri "$baseUrl/$userId" -Method Get
$response | ConvertTo-Json -Depth 3

Start-Sleep -Seconds 1

# Update user
$updatedUser = @{
    username = "updateduser"
    email = "updateduser@example.com"
    password = "NewPass456"
}
$updateJson = $updatedUser | ConvertTo-Json

Write-Host "Updating user ID $userId..."
$response = Invoke-RestMethod -Uri "$baseUrl/$userId" -Method Put -Body $updateJson -ContentType "application/json"
Write-Host "User updated." -ForegroundColor Yellow

Start-Sleep -Seconds 1

# Authenticate user
$auth = @{
    username = "updateduser"
    password = "NewPass456"
}
$authJson = $auth | ConvertTo-Json

Write-Host "Authenticating user..."
$response = Invoke-RestMethod -Uri "$baseUrl/authenticate" -Method Post -Body $authJson -ContentType "application/json"
Write-Host "Authentication status: Success" -ForegroundColor Cyan

Start-Sleep -Seconds 1

# Delete user
Write-Host "Deleting user ID $userId..."
Invoke-RestMethod -Uri "$baseUrl/$userId" -Method Delete
Write-Host "User deleted." -ForegroundColor Red
