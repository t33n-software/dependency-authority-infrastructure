variable "project_id" {
  description = "Google Cloud project ID that owns the repository. The organization instance supplies this value; the core never carries a default."
  type        = string
}

variable "location" {
  description = "Artifact Registry location of the repository."
  type        = string
}

variable "repository" {
  description = "Repository resource reference, for example the id output of the artifact-registry module."
  type        = string
}

variable "readers" {
  description = "Members receiving roles/artifactregistry.reader on this repository, for example consumer or verifier identities."
  type        = set(string)
  default     = []
}

variable "writers" {
  description = "Members receiving roles/artifactregistry.writer on this repository, for example the zone's lane identity. Evidence zones never grant routine delete authority through this module."
  type        = set(string)
  default     = []
}
