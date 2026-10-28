output "resource_group_name" {
  description = "Nombre del grupo de recursos creado"
  value       = azurerm_resource_group.main.name
}

output "vnet_id" {
  description = "ID de la red virtual creada"
  value       = azurerm_virtual_network.main.id
}

output "app_subnet_id" {
  description = "ID de la subred de aplicación creada"
  value       = azurerm_subnet.app.id
}

output "app_nsg_id" {
  description = "ID del grupo de seguridad de red creado"
  value       = azurerm_network_security_group.app.id
}