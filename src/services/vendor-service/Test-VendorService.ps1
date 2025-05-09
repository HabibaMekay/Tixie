
$baseUrl = "http://localhost:9060/vendors"


$vendor = @{
    name     = "TestVendor"
    email    = "test@example.com"
    password = "securepassword"
}
$vendorJson = $vendor | ConvertTo-Json -Depth 3

Write-Host "`nCreating vendor..."
$response = Invoke-RestMethod -Uri $baseUrl -Method Post -Body $vendorJson -ContentType "application/json" -ErrorAction SilentlyContinue

if ($response -ne $null -or $LASTEXITCODE -eq 0) {
    Write-Host " Vendor created."
} else {
    Write-Host " Failed to create vendor."
}

Start-Sleep -Seconds 1

Write-Host "`nGetting all vendors..."
$response = Invoke-RestMethod -Uri $baseUrl -Method Get -ErrorAction SilentlyContinue

if ($response) {
    $vendorId = $response[0].id
    Write-Host " Retrieved vendors. Using ID: $vendorId"
} else {
    Write-Host " Failed to retrieve vendors."
    exit
}

Start-Sleep -Seconds 1

Write-Host "`nGetting vendor by ID..."
$response = Invoke-RestMethod -Uri "$baseUrl/$vendorId" -Method Get -ErrorAction SilentlyContinue

if ($response) {
    Write-Host " Retrieved vendor with ID $vendorId"
} else {
    Write-Host " Failed to get vendor by ID"
}

Start-Sleep -Seconds 1

Write-Host "`nUpdating vendor..."
$updatedVendor = @{
    name  = "UpdatedVendor"
    email = "updated@example.com"
}
$updatedJson = $updatedVendor | ConvertTo-Json -Depth 3

$response = Invoke-RestMethod -Uri "$baseUrl/$vendorId" -Method Put -Body $updatedJson -ContentType "application/json" -ErrorAction SilentlyContinue

if ($?) {
    Write-Host " Vendor updated."
} else {
    Write-Host " Failed to update vendor."
}

Start-Sleep -Seconds 1

Write-Host "`nAuthenticating vendor..."
$auth = @{
    username = "UpdatedVendor"
    password = "securepassword"
}
$authJson = $auth | ConvertTo-Json -Depth 3

$response = Invoke-RestMethod -Uri "$baseUrl/authenticate" -Method Post -Body $authJson -ContentType "application/json" -ErrorAction SilentlyContinue

if ($?) {
    Write-Host " Vendor authenticated."
} else {
    Write-Host " Authentication failed."
}

Start-Sleep -Seconds 1

Write-Host "`nDeleting vendor..."
$response = Invoke-RestMethod -Uri "$baseUrl/$vendorId" -Method Delete -ErrorAction SilentlyContinue

if ($?) {
    Write-Host " Vendor deleted."
} else {
    Write-Host " Failed to delete vendor."
}
