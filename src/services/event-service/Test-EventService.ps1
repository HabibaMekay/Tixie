
$baseUrl = "http://localhost:8080/v1"

$event = @{
    name          = "Concert Night"
    location      = "Stadium"
    date          = "2025-06-01T19:00:00Z"
    total_tickets = 100
    vendor_id     = 1  
}
$eventJson = $event | ConvertTo-Json -Depth 3

Write-Host "`nCreating event..."
$response = Invoke-RestMethod -Uri $baseUrl -Method Post -Body $eventJson -ContentType "application/json" -ErrorAction SilentlyContinue

if ($response) {
    Write-Host "Event created."
    $eventId = $response.id
} else {
    Write-Host " Failed to create event."
    exit
}

Start-Sleep -Seconds 1

Write-Host "`nGetting all events..."
$response = Invoke-RestMethod -Uri $baseUrl -Method Get -ErrorAction SilentlyContinue

if ($response) {
    Write-Host " Retrieved events. Total: $($response.Count)"
} else {
    Write-Host " Failed to retrieve events."
}

Start-Sleep -Seconds 1

Write-Host "`nGetting event by ID..."
$response = Invoke-RestMethod -Uri "$baseUrl/$eventId" -Method Get -ErrorAction SilentlyContinue

if ($response) {
    Write-Host "Retrieved event: $($response.name)"
} else {
    Write-Host " Failed to retrieve event by ID"
}

Start-Sleep -Seconds 1

Write-Host "`nUpdating tickets sold..."
$ticketUpdate = @{
    tickets_to_buy = 5
}
$ticketJson = $ticketUpdate | ConvertTo-Json -Depth 2

$response = Invoke-RestMethod -Uri "$baseUrl/$eventId/tickets" -Method Patch -Body $ticketJson -ContentType "application/json" -ErrorAction SilentlyContinue

if ($response) {
    Write-Host " Tickets sold updated."
} else {
    Write-Host " Failed to update tickets sold."
}
