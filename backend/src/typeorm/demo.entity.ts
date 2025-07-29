import { Column, Entity, JoinColumn, JoinTable, ManyToMany, ManyToOne, OneToMany, PrimaryColumn, PrimaryGeneratedColumn, Unique } from 'typeorm';

@Entity()
export class Entry {
  @PrimaryGeneratedColumn({
    type: 'bigint',
    name: 'entry_id',
  })
  id: number;

  @Column({
    nullable: false,
    default: '',
  })
  title: string;
  
}
