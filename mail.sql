-- auto-generated definition
create table stats
(
    receiver    varchar(255) not null
        primary key,
    send_status int          null,
    send_time   datetime     null,
    receive_status int          null,
    receive_time   datetime     null
);