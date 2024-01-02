curl -XPUT -H "Content-Type: application/json" \
-u XXXXXXXX:XXXXXXX http://localhost:8094/api/index/title-index -d \
'{
  "name": "title-index",
  "type": "fulltext-index",
  "params": {
   "doc_config": {
    "docid_prefix_delim": "",
    "docid_regexp": "",
    "mode": "scope.collection.type_field",
    "type_field": "type"
   },
   "mapping": {
    "default_analyzer": "standard",
    "default_datetime_parser": "dateTimeOptional",
    "default_field": "_all",
    "default_mapping": {
     "dynamic": false,
     "enabled": false
    },
    "default_type": "_default",
    "docvalues_dynamic": false,
    "index_dynamic": false,
    "store_dynamic": false,
    "type_field": "_type",
    "types": {
     "0.items": {
      "dynamic": false,
      "enabled": true,
      "properties": {
       "amount": {
        "enabled": true,
        "dynamic": false,
        "fields": [
         {
          "docvalues": true,
          "include_in_all": true,
          "index": true,
          "name": "amount",
          "store": true,
          "type": "number"
         }
        ]
       },
       "bought": {
        "enabled": true,
        "dynamic": false,
        "fields": [
         {
          "index": true,
          "name": "bought",
          "store": true,
          "type": "boolean"
         }
        ]
       },
       "created": {
        "enabled": true,
        "dynamic": false,
        "fields": [
         {
          "docvalues": true,
          "include_in_all": true,
          "index": true,
          "name": "created",
          "store": true,
          "type": "number"
         }
        ]
       },
       "shop": {
        "enabled": true,
        "dynamic": false,
        "fields": [
         {
          "docvalues": true,
          "include_in_all": true,
          "include_term_vectors": true,
          "index": true,
          "name": "shop",
          "store": true,
          "type": "text"
         }
        ]
       },
       "title": {
        "enabled": true,
        "dynamic": false,
        "fields": [
         {
          "analyzer": "standard",
          "docvalues": true,
          "include_in_all": true,
          "include_term_vectors": true,
          "index": true,
          "name": "title",
          "store": true,
          "type": "text"
         }
        ]
       },
       "unit": {
        "enabled": true,
        "dynamic": false,
        "fields": [
         {
          "docvalues": true,
          "include_in_all": true,
          "include_term_vectors": true,
          "index": true,
          "name": "unit",
          "store": true,
          "type": "text"
         }
        ]
       },
       "updated": {
        "enabled": true,
        "dynamic": false,
        "fields": [
         {
          "docvalues": true,
          "include_in_all": true,
          "index": true,
          "name": "updated",
          "store": true,
          "type": "number"
         }
        ]
       }
      }
     }
    }
   },
   "store": {
    "indexType": "scorch",
    "segmentVersion": 15
   }
  },
  "sourceType": "gocbcore",
  "sourceName": "default",
  "sourceUUID": "1991cacdfeb09b75b98327df24a53df7",
  "sourceParams": {},
  "planParams": {
   "maxPartitionsPerPIndex": 1024,
   "indexPartitions": 1,
   "numReplicas": 0
  },
  "uuid": "7206c9973c1fb249"
 }'