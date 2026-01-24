data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "./cmd/loader",
  ]
}
env "gorm" {
  src = data.external_schema.gorm.url
  dev = "sqlite://file?mode=memory"
  migration {
    dir = "file://migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}