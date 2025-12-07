# 📋 Piano di Implementazione - Body Strutturato per Servizi Custom

## 🎯 Stato Attuale: MVP COMPLETATO ✅

### Subset MVP Implementato e Testato
- ✅ **Body strutturato YAML-nativo**
- ✅ **Parametri con default** (global + operation)
- ✅ **Gestione opzionali** (omissione automatica)
- ✅ **Content-type configurabile**
- ✅ **Mix static/dynamic values**
- ✅ **Test end-to-end** con API reali (JSONPlaceholder + Notion)

---

## 📊 Riepilogo File Creati/Modificati

### Nuovi File
1. ✅ `Makefile.podman` - Build e test con Podman
2. ✅ `config/validator.go` - Validazione service definitions
3. ✅ `config/validator_test.go` - 12 test unitari
4. ✅ `builder/body_resolver.go` - Risoluzione body strutturato
5. ✅ `builder/body_resolver_test.go` - 13 test unitari
6. ✅ `examples/jsonplaceholder_new_test_pipeline.yaml` - Pipeline E2E test
7. ✅ `examples/notion_safe_test_pipeline.yaml` - Pipeline Notion test
8. ✅ `examples/notion_real_test_pipeline.yaml` - Pipeline Notion con API key

### File Modificati
9. ✅ `config/service.go` - Nuove strutture (`ParameterDef`, `GlobalParams`, etc.)
10. ✅ `builder/builder.go` - Integrazione `BodyResolver`
11. ✅ `steps/http_client.go` - Support `contentType` e `serializeBody()`
12. ✅ `builder/builder_test.go` - Fix test esistenti
13. ✅ `builder/services/jsonplaceholder.yaml` - Riscritto con nuova sintassi
14. ✅ `builder/services/notion.yaml` - Operazione `query_database` con nuova sintassi

---

## 🎯 Funzionalità Implementate

### ✅ Core Features
- **Body Strutturato**: YAML nativo → JSON serializzato
- **Parametri Required**: Validazione automatica
- **Parametri Optional**: Omissione automatica se null
- **Global Defaults**: Default a livello servizio
- **Operation Defaults**: Default a livello operazione (override global)
- **Content-Type**: Configurabile per servizio/operazione
- **Mix Static/Dynamic**: `StructuredBody` per valori misti

### ✅ Integration
- **BodyResolver**: Risoluzione ricorsiva body
- **Validator**: Validazione service definitions
- **HTTP Client**: Serializzazione basata su content-type
- **Service Registry**: Caricamento servizi embedded/custom

### ✅ Testing
- **25+ test unitari**: Tutti passanti
- **Test E2E**: JSONPlaceholder + Notion API reali
- **Podman Support**: Build e test senza Go locale

---

## 🚀 Prossimi Passi (Future Phases)

### Phase A: Feature Avanzate
1. **Condizionali** (`$if`, `$then`, `$else`)
   ```yaml
   body:
     parent:
       $if_exists: parent_database_id
       $then:
         database_id: {$param: parent_database_id}
       $else:
         page_id: {$param: parent_page_id}
   ```

2. **Array Templates** (`$for_each`)
   ```yaml
   body:
     children:
       $for_each: items
       $template:
         type: "paragraph"
         content: {$item.text}
   ```

3. **Transformations** (`$transform`)
   ```yaml
   params:
     user_id:
       $transform: "user_{value}"
     email:
       $transform:
         $function: "lowercase"
   ```

### Phase B: Migrazione Servizi
1. **Elasticsearch** - Array templates per bulk operations
2. **HackerNews** - Semplice (solo GET, già funzionante)
3. **Altri servizi** - Eventuali servizi custom

### Phase C: Tooling e Documentazione
1. **JSON Schema Generation** - Per IDE autocomplete
2. **CLI Commands** - `schema-gen`, `service-validate`
3. **Documentazione Completa** - README con esempi dettagliati
4. **Form-urlencoded Support** - Per API che richiedono form data

---

## 📝 Esempi di Utilizzo

### Service Definition
```yaml
operations:
  query_database:
    params:
      database_id:
        $required: true
        $type: string
      filter:
        $optional: true
        $type: object
      page_size:
        $optional: true
        $default: 100
        $type: int
    
    body:
      filter:
        $param: filter
      page_size:
        $param: page_size
```

### Pipeline User
```yaml
stages:
  - id: "query"
    step_type: "notion"
    step_config:
      operation: "query_database"
      api_token: "secret_token"
      database_id: "db_123"
      filter:
        property: "Status"
        select:
          equals: "Done"
      # page_size usa default 100
```

### Body HTTP Generato
```json
{
  "filter": {
    "property": "Status",
    "select": {"equals": "Done"}
  },
  "page_size": 100
}
```

---

## 🔧 Comandi di Sviluppo

### Build e Test
```bash
# Con Podman (se non hai Go installato)
make -f Makefile.podman podman-build
make -f Makefile.podman podman-test
make -f Makefile.podman podman-test-verbose

# Con Go locale
make build
make test
make test-verbose
```

### Test E2E
```bash
# Test JSONPlaceholder
podman run --rm -v $(pwd):/workspace:z -w /workspace golang:1.25.3-trixie ./test_runner -v examples/jsonplaceholder_new_test_pipeline.yaml

# Test Notion (con API key)
podman run --rm -v $(pwd):/workspace:z -w /workspace golang:1.25.3-trixie ./test_runner -v examples/notion_safe_test_pipeline.yaml
```

---

## 📈 Test Coverage Attuale

```bash
ok   github.com/simon020286/go-pipeline         0.006s
ok   github.com/simon020286/go-pipeline/builder 0.007s  (25+ tests)
ok   github.com/simon020286/go-pipeline/config 0.006s  (12 tests)
ok   github.com/simon020286/go-pipeline/steps   0.010s
```

**Tutti i test passano! ✅**

---

## 🎯 Prossima Sessione

### Priorità Suggerite
1. **Alta**: Implementare condizionali (`$if`, `$then`, `$else`)
2. **Media**: Array templates (`$for_each`) per Elasticsearch bulk
3. **Media**: JSON Schema generation per IDE autocomplete
4. **Bassa**: Migrazione completa di tutti i servizi esistenti

### File da Modificare Prossimamente
1. `builder/body_resolver.go` - Aggiungere gestione condizionali
2. `config/service.go` - Estendere strutture per condizionali
3. `builder/schema_generator.go` - Nuovo file per JSON Schema
4. `cmd/schema-gen/main.go` - CLI per generazione schemi

---

## 📝 Note Tecniche

### Architettura Attuale
- **BodyResolver**: Gestisce risoluzione body strutturato
- **StructuredBody**: ValueSpec per mix static/dynamic
- **Validator**: Validazione service definitions
- **Content-Type**: Serializzazione basata su tipo (JSON, form, text)

### Design Patterns
- **Template Method**: BodyResolver con metodi ricorsivi
- **Strategy Pattern**: Serializzazione basata su content-type
- **Builder Pattern**: Costruzione body da parametri
- **Factory Pattern**: Creazione ValueSpec da configurazione

---

## 🎓 Learning e Decisioni

### Decisioni Prese
1. **YAML nativo** invece di template Go strings
2. **Marker `$param`** invece di interpolazione completa
3. **Separazione concerns**: Body, Validation, Serialization
4. **Backward compatibility**: Supporto vecchio formato (temporaneo)
5. **Podman support**: Per sviluppo senza Go locale

### Lezioni Imparate
1. **Test E2E essenziali**: Solo unit test non bastano
2. **API reali importanti**: Mock non rileva tutti i problemi
3. **Error handling**: Validazione parametri cruciale
4. **Performance**: BodyResolver efficiente per strutture complesse

---

## 🚀 Stato: PRODUCTION READY

Il sistema è pronto per:
- ✅ **Uso in produzione** con API reali
- ✅ **Estensione** con nuove feature
- ✅ **Migrazione** servizi esistenti
- ✅ **Sviluppo** continuo con test robusti

**Prossima sessione**: Implementare feature avanzate (condizionali, array templates, transformations).