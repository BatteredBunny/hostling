data "external_schema" "gorm_sqlite" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "./cmd/loader",
    "-dialect=sqlite",
  ]
}

data "external_schema" "gorm_postgres" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "./cmd/loader",
    "-dialect=postgres",
  ]
}
env "sqlite" {
  src = data.external_schema.gorm_sqlite.url
  dev = "sqlite://file?mode=memory"
  migration {
    dir = "file://migrations/sqlite"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
env "postgresql" {
  src = data.external_schema.gorm_postgres.url
  dev = "docker://postgres/15/dev?search_path=public"
  migration {
    dir = "file://migrations/postgresql"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}