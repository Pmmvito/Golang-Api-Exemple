# Golang API de Oportunidades de Emprego

Uma **API REST** moderna e eficiente para gerenciar oportunidades de emprego, desenvolvida em **Go (Golang)** utilizando as melhores práticas e tecnologias mais recentes do ecossistema.

## 🚀 Sobre o Projeto

Esta API permite o **CRUD completo** (Create, Read, Update, Delete) de vagas de emprego, oferecendo endpoints para:

- ✅ Criar novas oportunidades de emprego
- ✅ Listar todas as vagas disponíveis
- ✅ Visualizar detalhes de uma vaga específica
- ✅ Atualizar informações de vagas existentes
- ✅ Remover vagas do sistema

## 🛠️ Tecnologias Utilizadas

- **[Go](https://golang.org/)** - Linguagem de programação
- **[Gin](https://gin-gonic.com/)** - Framework web HTTP
- **[GORM](https://gorm.io/)** - ORM (Object Relational Mapping)
- **[SQLite](https://sqlite.org/)** - Banco de dados
- **[Swaggo](https://github.com/swaggo/swag)** - Geração automática de documentação Swagger
- **[Swagger UI](https://swagger.io/tools/swagger-ui/)** - Interface de documentação interativa

## 📋 Pré-requisitos

- Go 1.21 ou superior
- GCC (para SQLite driver) ou uso de driver SQLite puro Go

## 🚀 Como Executar

### 1. Clone o repositório
```bash
git clone https://github.com/Pmmvito/Golang-Api-Exemple.git
cd Golang-Api-Exemple
```

### 2. Instale as dependências
```bash
go mod tidy
```

### 3. Execute a aplicação
```bash
go run main.go
```

A API estará disponível em: `http://localhost:8080`

## 📚 Documentação da API

### Swagger UI
A documentação interativa da API está disponível em:
```
http://localhost:8080/swagger/index.html
```

### Endpoints Principais

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| `GET` | `/api/v1/opening` | Buscar uma vaga específica |
| `GET` | `/api/v1/openings` | Listar todas as vagas |
| `POST` | `/api/v1/opening` | Criar nova vaga |
| `PUT` | `/api/v1/opening` | Atualizar vaga existente |
| `DELETE` | `/api/v1/opening` | Remover vaga |

### Exemplo de Payload (Criar Vaga)
```json
{
  "role": "Desenvolvedor Backend",
  "company": "Tech Corp",
  "location": "São Paulo, SP",
  "remote": true,
  "link": "https://example.com/job",
  "salary": 8000
}
```

## 🏗️ Estrutura do Projeto

```
├── config/          # Configurações (DB, Logger)
├── handler/         # Handlers HTTP (Controllers)
├── router/          # Configuração de rotas
├── schemas/         # Modelos de dados
├── docs/            # Documentação Swagger gerada
├── db/              # Banco de dados SQLite
├── main.go          # Ponto de entrada da aplicação
└── README.md        # Documentação do projeto
```

## 📖 Gerando Documentação Swagger

Este projeto utiliza **Swaggo** para gerar automaticamente a documentação Swagger a partir de comentários no código.

### Instalação do Swag CLI
```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

### Gerar documentação
```bash
swag init
```

### Comentários Swagger nos Handlers
Os endpoints são documentados usando comentários especiais:

```go
// CreateOpening godoc
// @Summary      Criar nova vaga
// @Description  Cria uma nova oportunidade de emprego
// @Tags         openings
// @Accept       json
// @Produce      json
// @Param        request  body     CreateOpeningRequest  true  "Request body"
// @Success      200      {object} CreateOpeningResponse
// @Failure      400      {object} ErrorResponse
// @Router       /opening [post]
```

## 🔗 Links

- **Repositório GitHub**: [https://github.com/Pmmvito/Golang-Api-Exemple](https://github.com/Pmmvito/Golang-Api-Exemple)
- **Documentação Swagger**: `http://localhost:8080/swagger/index.html` (quando rodando localmente)
- **Swagger JSON**: `http://localhost:8080/swagger/doc.json`

## 🤝 Contribuindo

1. Faça um fork do projeto
2. Crie uma branch para sua feature (`git checkout -b feature/AmazingFeature`)
3. Commit suas mudanças (`git commit -m 'Add some AmazingFeature'`)
4. Push para a branch (`git push origin feature/AmazingFeature`)
5. Abra um Pull Request

## 📝 Licença

Este projeto está sob a licença MIT. Veja o arquivo [LICENSE](LICENSE) para mais detalhes.

## 👨‍💻 Autor

**Vitor Benevento** - [GitHub](https://github.com/Pmmvito)

---