import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Entry } from './typeorm';
import { Repository } from 'typeorm';

@Injectable()
export class AppService {
  constructor(
    @InjectRepository(Entry) private readonly EntriesRepository: Repository<Entry>,
  ) {}

  async getHello(): Promise<Entry[]> {
    return await this.EntriesRepository.createQueryBuilder("*").getMany();
  }

  async createEntry(title: string) {
        const newEntry = this.EntriesRepository.create({
            title: title,
        });

        const savedBlind = await this.EntriesRepository.save(newEntry);

        return this.getHello()
  }
}
