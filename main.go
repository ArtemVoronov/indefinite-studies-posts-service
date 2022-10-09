package main

import (
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/app"
)

func main() {
	app.Start()

	// TODO:
	// 1. add 'uuid' to post (for consistent hashing and rendering on UI, seraching etc)
	// 2. add shards and solid_shards tables to model:
	// shards: id, db_connection_url, bucked_index_start, bucket_index_end
	// solid_shards: id, shard_name, db_connection_url
	// 3. add murmur3 utils for getting bucket_index based on post 'uuid' attribute (go get "github.com/spaolacci/murmur3")
	// 4. add queries for getting db_connection_url based on bucket_index: select from shards where @bucket_index >= bucked_index_start and @bucket_index < bucket_index_end
	// 5. add processing create, update, delete operations based on appropriate db_connection_url
	// 6. add processing getting all posts from all databases with pagination
	// 7. refactor feed builder service if needs
}
