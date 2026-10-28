variable "subscription_id" {
  description = ""
  type        = string
  sensitive   = true
}

variable "location" {
  description = "Región de despliegue de los recursos"
  type        = string
  default     = "brazilsouth"
}

variable "environment" {
  description = "Variables de entorno del despliegue"
  type        = string
  default     = "lab"
  validation {
    condition     = contains(["lab", "dev", "prod"], var.environment)
    error_message = "El entorno debe ser lab, dev o prod."
  }
}

variable "project_name" {
  description = "Nombre identificador del proyecto, utilizado para nombrar el grupo de recursos"
  type        = string
  default     = "azure-lab"
}

variable "vnet_adress_space" {
  description = "Dirección de red virtual"
  type        = list(string)
  default     = ["10.234.0.0/16"]
}

variable "app_subnet_prefix" {
  description = "Prefijo de subred para la aplicación"
  type        = string
  default     = "10.234.1.0/24"
}

variable "admin_public_ip" {
  description = "Dirección IP pública del administrador"
  type        = string
  validation {
    condition     = can(cidrhost(var.admin_public_ip, 0)) && (endswith(var.admin_public_ip, "/32"))
    error_message = "La dirección IP pública del administrador debe ser una dirección válida en la red con prefijo /32."
  }
}
