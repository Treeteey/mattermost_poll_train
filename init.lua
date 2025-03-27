box.cfg{
    listen = 3301,
    work_dir = '/var/lib/tarantool',
    log_level = 5
}

-- Создание пространства для голосований
box.schema.space.create('votes', {
    format = {
        {name = 'id', type = 'string'},
        {name = 'creator_id', type = 'string'},
        {name = 'question', type = 'string'},
        {name = 'options', type = 'array'},
        {name = 'votes', type = 'map'},
        {name = 'is_active', type = 'boolean'},
        {name = 'channel_id', type = 'string'}
    }
})

-- Создание индекса по ID голосования
box.space.votes:create_index('primary', {
    parts = {'id'},
    unique = true
})

-- Создание индекса по ID канала
box.space.votes:create_index('channel_id', {
    parts = {'channel_id'},
    unique = false
})

-- Создание пользователя для доступа
box.schema.user.create('admin', {
    password = 'admin',
    if_not_exists = true
})

-- Предоставление прав на пространство
box.schema.user.grant('admin', 'read,write,execute', 'space', 'votes') 